#pragma once

#include <cstdint>
#include <string>

namespace iris {

struct User {
    std::string id;
    std::string username;
    int64_t created_at   = 0;
    int64_t updated_at   = 0;
};

} // namespace iris
