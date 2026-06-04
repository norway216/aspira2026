#pragma once

#include "Error.h"

#include <variant>
#include <optional>
#include <stdexcept>

template <typename T>
class Result {
public:
    Result(T value) : m_data(std::move(value)) {}
    Result(Error error) : m_data(std::move(error)) {}

    static Result ok(T value) { return Result(std::move(value)); }
    static Result fail(ErrorCode code, QString msg) {
        return Result(Error::fail(code, std::move(msg)));
    }

    bool isOk() const { return std::holds_alternative<T>(m_data); }
    bool isFail() const { return std::holds_alternative<Error>(m_data); }
    explicit operator bool() const { return isOk(); }

    T& value() {
        if (isFail()) throw std::logic_error("Result: accessing value of failed result");
        return std::get<T>(m_data);
    }
    const T& value() const {
        if (isFail()) throw std::logic_error("Result: accessing value of failed result");
        return std::get<T>(m_data);
    }

    T valueOr(T defaultVal) const {
        if (isOk()) return std::get<T>(m_data);
        return defaultVal;
    }

    Error& error() {
        if (isOk()) throw std::logic_error("Result: accessing error of successful result");
        return std::get<Error>(m_data);
    }
    const Error& error() const {
        if (isOk()) throw std::logic_error("Result: accessing error of successful result");
        return std::get<Error>(m_data);
    }

    std::optional<T> toOptional() const {
        if (isOk()) return std::get<T>(m_data);
        return std::nullopt;
    }

    template <typename F>
    auto map(F&& f) -> Result<decltype(f(std::declval<T>()))> {
        if (isOk()) return Result<decltype(f(std::declval<T>()))>::ok(f(std::get<T>(m_data)));
        return Result<decltype(f(std::declval<T>()))>::fail(std::get<Error>(m_data).code, std::get<Error>(m_data).message);
    }

private:
    std::variant<T, Error> m_data;
};

// Void Result specialization
template <>
class Result<void> {
public:
    Result() : m_error(Error::ok()) {}
    Result(Error error) : m_error(std::move(error)) {}

    static Result ok() { return Result(); }
    static Result fail(ErrorCode code, QString msg) {
        return Result(Error::fail(code, std::move(msg)));
    }

    bool isOk() const { return m_error.isOk(); }
    bool isFail() const { return m_error.isFail(); }
    explicit operator bool() const { return isOk(); }

    Error& error() { return m_error; }
    const Error& error() const { return m_error; }

private:
    Error m_error;
};
