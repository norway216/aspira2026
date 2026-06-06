#pragma once

#include <string>
#include <cstdint>
#include <chrono>
#include <optional>
#include <vector>
#include <atomic>
#include <sstream>
#include <cstring>
#include <algorithm>
#include <cctype>

namespace payment_engine {

// ============================================================
// Enums and Core Structures
// ============================================================

enum class TxnStatus { Pending, Processing, Completed, Failed, Refunded };
enum class ErrorCode {
    None = 0,
    InsufficientFunds,
    AccountNotFound,
    AccountFrozen,
    InvalidCurrency,
    RateNotFound,
    LimitExceeded,
    InternalError,
    InvalidRequest
};

struct TransactionRequest {
    std::string transaction_id, merchant_id, payer_account_id, payee_account_id;
    std::string source_currency, target_currency;
    int64_t source_amount = 0, fee = 0;
    std::string reference_id, timestamp;
};

struct TransactionResult {
    std::string transaction_id;
    int64_t target_amount = 0;
    double exchange_rate = 0.0;
    int64_t fee = 0;
    TxnStatus status = TxnStatus::Pending;
    std::string hash_chain_current, processed_at;
};

struct EngineError {
    ErrorCode code = ErrorCode::None;
    std::string message;
    bool retriable = false;
};

struct EngineRequest {
    std::string id, type;
    std::string payload_json;
    int session_fd = -1;
};

struct EngineResponse {
    std::string id, status;
    std::string payload_json;
    std::optional<EngineError> error;
    int session_fd = -1;
};

using Clock = std::chrono::steady_clock;

// ============================================================
// Minimal JSON Parser (no external dependencies)
// ============================================================

class SimpleJson {
public:
    SimpleJson() = default;
    explicit SimpleJson(std::string json) : raw_(std::move(json)) {}

    std::optional<std::string> GetString(const std::string& key) const {
        auto pos = findKey(key);
        if (!pos) return std::nullopt;
        return extractString(*pos);
    }

    std::optional<int64_t> GetInt64(const std::string& key) const {
        auto pos = findKey(key);
        if (!pos) return std::nullopt;
        auto val_str = extractNumber(*pos);
        if (!val_str) return std::nullopt;
        return std::stoll(*val_str);
    }

    std::optional<double> GetDouble(const std::string& key) const {
        auto pos = findKey(key);
        if (!pos) return std::nullopt;
        auto val_str = extractNumber(*pos);
        if (!val_str) return std::nullopt;
        return std::stod(*val_str);
    }

    std::optional<std::string> GetObject(const std::string& key) const {
        auto pos = findKey(key);
        if (!pos) return std::nullopt;
        return extractObject(*pos);
    }

    const std::string& Raw() const { return raw_; }

    static std::string Escape(const std::string& s) {
        std::string out;
        out.reserve(s.size() + 2);
        for (char c : s) {
            if (c == '"' || c == '\\') out.push_back('\\');
            out.push_back(c);
        }
        return out;
    }

    static std::string BuildObject(const std::vector<std::pair<std::string, std::string>>& fields) {
        std::string out = "{";
        for (size_t i = 0; i < fields.size(); ++i) {
            if (i > 0) out += ",";
            out += "\"" + Escape(fields[i].first) + "\":" + fields[i].second;
        }
        out += "}";
        return out;
    }

    static std::string BuildStringField(const std::string& key, const std::string& value) {
        return "\"" + Escape(key) + "\":\"" + Escape(value) + "\"";
    }

    static std::string BuildIntField(const std::string& key, int64_t value) {
        return "\"" + Escape(key) + "\":" + std::to_string(value);
    }

    static std::string BuildDoubleField(const std::string& key, double value) {
        std::ostringstream oss;
        oss << "\"" << Escape(key) << "\":" << value;
        return oss.str();
    }

    static std::string BuildObjectField(const std::string& key, const std::string& object_json) {
        return "\"" + Escape(key) + "\":" + object_json;
    }

    static std::string ToObject(const std::vector<std::string>& field_strings) {
        std::string out = "{";
        for (size_t i = 0; i < field_strings.size(); ++i) {
            if (i > 0) out += ",";
            out += field_strings[i];
        }
        out += "}";
        return out;
    }

    static TransactionRequest ParseTransactionRequest(const std::string& json) {
        SimpleJson doc(json);
        TransactionRequest req;
        if (auto v = doc.GetString("transaction_id")) req.transaction_id = *v;
        if (auto v = doc.GetString("merchant_id")) req.merchant_id = *v;
        if (auto v = doc.GetString("payer_account_id")) req.payer_account_id = *v;
        if (auto v = doc.GetString("payee_account_id")) req.payee_account_id = *v;
        if (auto v = doc.GetString("source_currency")) req.source_currency = *v;
        if (auto v = doc.GetString("target_currency")) req.target_currency = *v;
        if (auto v = doc.GetString("reference_id")) req.reference_id = *v;
        if (auto v = doc.GetString("timestamp")) req.timestamp = *v;
        if (auto i = doc.GetInt64("source_amount")) req.source_amount = *i;
        if (auto i = doc.GetInt64("fee")) req.fee = *i;
        return req;
    }

    static std::string SerializeTransactionResult(const TransactionResult& res) {
        std::vector<std::string> fields;
        fields.push_back(BuildStringField("transaction_id", res.transaction_id));
        fields.push_back(BuildIntField("target_amount", res.target_amount));
        fields.push_back(BuildDoubleField("exchange_rate", res.exchange_rate));
        fields.push_back(BuildIntField("fee", res.fee));
        std::string status_str;
        switch (res.status) {
            case TxnStatus::Pending:    status_str = "Pending"; break;
            case TxnStatus::Processing: status_str = "Processing"; break;
            case TxnStatus::Completed:  status_str = "Completed"; break;
            case TxnStatus::Failed:     status_str = "Failed"; break;
            case TxnStatus::Refunded:   status_str = "Refunded"; break;
        }
        fields.push_back(BuildStringField("status", status_str));
        fields.push_back(BuildStringField("hash_chain_current", res.hash_chain_current));
        fields.push_back(BuildStringField("processed_at", res.processed_at));
        return ToObject(fields);
    }

    static std::string SerializeError(const EngineError& err) {
        std::vector<std::string> fields;
        fields.push_back(BuildIntField("code", static_cast<int>(err.code)));
        fields.push_back(BuildStringField("message", err.message));
        fields.push_back(err.retriable ? BuildStringField("retriable", "true")
                                        : BuildStringField("retriable", "false"));
        return ToObject(fields);
    }

private:
    std::string raw_;

    void skipWhitespace(size_t& pos) const {
        while (pos < raw_.size() && (raw_[pos] == ' ' || raw_[pos] == '\t' ||
               raw_[pos] == '\n' || raw_[pos] == '\r'))
            ++pos;
    }

    std::optional<size_t> findKey(const std::string& key) const {
        std::string target = "\"" + key + "\"";
        size_t pos = 0;
        while (true) {
            pos = raw_.find(target, pos);
            if (pos == std::string::npos) return std::nullopt;
            if (pos > 0 && raw_[pos - 1] == '\\') {
                pos += target.size();
                continue;
            }
            pos += target.size();
            skipWhitespace(pos);
            if (pos < raw_.size() && raw_[pos] == ':') {
                ++pos;
                skipWhitespace(pos);
                return pos;
            }
        }
    }

    std::optional<std::string> extractString(size_t pos) const {
        if (pos >= raw_.size() || raw_[pos] != '"') return std::nullopt;
        ++pos;
        std::string result;
        bool escaped = false;
        while (pos < raw_.size()) {
            char c = raw_[pos++];
            if (escaped) {
                result += c;
                escaped = false;
            } else if (c == '\\') {
                escaped = true;
            } else if (c == '"') {
                return result;
            } else {
                result += c;
            }
        }
        return std::nullopt;
    }

    std::optional<std::string> extractNumber(size_t pos) const {
        if (pos >= raw_.size()) return std::nullopt;
        if (raw_[pos] == '"') return extractString(pos);
        size_t start = pos;
        if (raw_[pos] == '-') ++pos;
        while (pos < raw_.size() && (std::isdigit(raw_[pos]) ||
               raw_[pos] == '.' || raw_[pos] == 'e' || raw_[pos] == 'E' ||
               raw_[pos] == '+' || raw_[pos] == '-')) {
            ++pos;
        }
        if (pos == start) return std::nullopt;
        return raw_.substr(start, pos - start);
    }

    std::optional<std::string> extractObject(size_t pos) const {
        if (pos >= raw_.size() || raw_[pos] != '{') return std::nullopt;
        int depth = 0;
        size_t start = pos;
        while (pos < raw_.size()) {
            if (raw_[pos] == '{') ++depth;
            else if (raw_[pos] == '}') {
                --depth;
                if (depth == 0) return raw_.substr(start, pos - start + 1);
            } else if (raw_[pos] == '"') {
                ++pos;
                while (pos < raw_.size()) {
                    if (raw_[pos] == '\\') pos += 2;
                    else if (raw_[pos] == '"') break;
                    else ++pos;
                }
            }
            ++pos;
        }
        return std::nullopt;
    }
};

// ============================================================
// Utility: timestamp helpers
// ============================================================

inline std::string CurrentTimestampISO() {
    auto now = std::chrono::system_clock::now();
    auto tt = std::chrono::system_clock::to_time_t(now);
    auto ms = std::chrono::duration_cast<std::chrono::milliseconds>(
                  now.time_since_epoch()) % 1000;
    std::tm tm;
    gmtime_r(&tt, &tm);
    char buf[64];
    std::strftime(buf, sizeof(buf), "%Y-%m-%dT%H:%M:%S", &tm);
    std::snprintf(buf + strlen(buf), sizeof(buf) - strlen(buf), ".%03lldZ",
                  (long long)ms.count());
    return std::string(buf);
}

inline int64_t CurrentTimestampNS() {
    return std::chrono::duration_cast<std::chrono::nanoseconds>(
               std::chrono::system_clock::now().time_since_epoch()).count();
}

} // namespace payment_engine
