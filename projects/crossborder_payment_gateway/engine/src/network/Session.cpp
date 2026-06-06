#include "network/Session.h"
#include "network/TCPServer.h"

#include <unistd.h>
#include <sys/socket.h>
#include <cerrno>
#include <cstring>
#include <iostream>
#include <arpa/inet.h>

namespace payment_engine {

Session::Session(int fd, TCPServer* server)
    : fd_(fd)
    , server_(server)
{
}

Session::~Session() {
    Stop();
}

void Session::Start() {
    active_.store(true, std::memory_order_release);

    read_thread_ = std::thread([this]() { readLoop(); });
    write_thread_ = std::thread([this]() { writeLoop(); });
}

void Session::Stop() {
    bool expected = true;
    if (!active_.compare_exchange_strong(expected, false, std::memory_order_acq_rel)) {
        return; // Already stopped
    }

    // Shutdown the socket to unblock reads/writes
    if (fd_ >= 0) {
        ::shutdown(fd_, SHUT_RDWR);
    }

    if (read_thread_.joinable()) {
        read_thread_.join();
    }
    if (write_thread_.joinable()) {
        write_thread_.join();
    }

    if (fd_ >= 0) {
        ::close(fd_);
        fd_ = -1;
    }

    std::cout << "[Session] Session " << fd_ << " stopped" << std::endl;
}

bool Session::readExact(uint8_t* buf, size_t n) {
    size_t total = 0;
    while (total < n) {
        ssize_t r = ::read(fd_, buf + total, n - total);
        if (r > 0) {
            total += r;
        } else if (r == 0) {
            return false; // EOF
        } else {
            if (errno == EINTR) continue;
            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                // Non-blocking, try again after a short yield
                std::this_thread::yield();
                continue;
            }
            return false; // Error
        }
    }
    return true;
}

bool Session::writeExact(const uint8_t* buf, size_t n) {
    size_t total = 0;
    while (total < n) {
        ssize_t w = ::write(fd_, buf + total, n - total);
        if (w > 0) {
            total += w;
        } else if (w == 0) {
            return false;
        } else {
            if (errno == EINTR) continue;
            if (errno == EAGAIN || errno == EWOULDBLOCK) {
                // Non-blocking: yield and retry
                std::this_thread::yield();
                continue;
            }
            return false; // Error
        }
    }
    return true;
}

void Session::readLoop() {
    while (active_.load(std::memory_order_acquire)) {
        // Read 4-byte big-endian length prefix
        uint8_t len_buf[4];
        if (!readExact(len_buf, 4)) {
            break; // Connection closed or error
        }

        // Parse big-endian length
        uint32_t json_len = (static_cast<uint32_t>(len_buf[0]) << 24) |
                            (static_cast<uint32_t>(len_buf[1]) << 16) |
                            (static_cast<uint32_t>(len_buf[2]) << 8) |
                             static_cast<uint32_t>(len_buf[3]);

        if (json_len == 0 || json_len > 64 * 1024) {
            // Invalid length, close connection
            break;
        }

        // Read the JSON payload
        std::string json(json_len, '\0');
        if (!readExact(reinterpret_cast<uint8_t*>(&json[0]), json_len)) {
            break;
        }

        // Parse the outer envelope
        SimpleJson doc(json);
        EngineRequest req;
        req.id = doc.GetString("id").value_or("");
        req.type = doc.GetString("type").value_or("");
        req.session_fd = fd_;

        // Extract the payload object as raw JSON
        auto payload = doc.GetObject("payload");
        if (payload) {
            req.payload_json = *payload;
        } else {
            req.payload_json = json; // Use whole message as payload
        }

        // Push to the shared request queue
        if (server_) {
            if (!server_->PushRequest(std::move(req))) {
                // Queue full, should not happen in normal operation
                std::cerr << "[Session] Request queue full, dropping request from fd "
                          << fd_ << std::endl;
            }
        }
    }

    // Clean exit
    active_.store(false, std::memory_order_release);
    Stop();
}

void Session::writeLoop() {
    while (active_.load(std::memory_order_acquire)) {
        auto maybe_msg = output_queue_.tryPop();
        if (!maybe_msg) {
            // Queue empty, yield briefly
            std::this_thread::yield();
            continue;
        }

        const std::string& resp_json = *maybe_msg;

        // Write 4-byte big-endian length prefix
        uint32_t len = static_cast<uint32_t>(resp_json.size());
        uint8_t len_buf[4];
        len_buf[0] = static_cast<uint8_t>((len >> 24) & 0xFF);
        len_buf[1] = static_cast<uint8_t>((len >> 16) & 0xFF);
        len_buf[2] = static_cast<uint8_t>((len >> 8) & 0xFF);
        len_buf[3] = static_cast<uint8_t>(len & 0xFF);

        if (!writeExact(len_buf, 4)) {
            break;
        }

        // Write JSON payload
        if (!writeExact(reinterpret_cast<const uint8_t*>(resp_json.data()),
                        resp_json.size())) {
            break;
        }
    }

    active_.store(false, std::memory_order_release);
}

bool Session::SendResponse(const EngineResponse& resp) {
    // Serialize EngineResponse to JSON
    std::vector<std::string> fields;
    fields.push_back(SimpleJson::BuildStringField("id", resp.id));
    fields.push_back(SimpleJson::BuildStringField("status", resp.status));

    if (!resp.payload_json.empty()) {
        fields.push_back(SimpleJson::BuildObjectField("payload", resp.payload_json));
    }

    if (resp.error.has_value()) {
        fields.push_back(SimpleJson::BuildObjectField("error",
                          SimpleJson::SerializeError(*resp.error)));
    }

    std::string json = SimpleJson::ToObject(fields);

    // Push to SPSC output queue
    return output_queue_.tryPush(std::move(json));
}

} // namespace payment_engine
