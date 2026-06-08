// RK3588 Recovery — Unit Tests
#include "recovery/Partition.h"
#include "recovery/BackupEngine.h"
#include "recovery/ChecksumVerifier.h"
#include "recovery/Logger.h"
#include <iostream>
#include <cstring>
#include <cassert>

using namespace rk3588::recovery;

static int tests_passed = 0;
static int tests_failed = 0;

#define TEST(name) std::cout << "  " << name << "... ";
#define PASS()      std::cout << "PASSED\n"; tests_passed++
#define FAIL(msg)   std::cout << "FAILED: " << msg << "\n"; tests_failed++

void test_partition_manager() {
    TEST("Create simulated partitions");
    PartitionManager pm;
    bool ok = pm.create_simulated_partition("test_sys", "/tmp/test_sys.img",
                                             100*1024*1024, "ext4");
    assert(ok);
    PASS();

    TEST("Get partition info");
    auto* p = pm.get_partition("test_sys");
    assert(p != nullptr);
    assert(p->size_bytes == 100*1024*1024);
    PASS();

    TEST("Free space check");
    assert(pm.has_enough_space("test_sys", 50*1024*1024));
    PASS();
}

void test_checksum_verifier() {
    TEST("SHA256 compute");
    ChecksumVerifier cv;
    // Create test file
    const char* data = "Hello RK3588 Recovery Test Data!";
    {
        FILE* f = fopen("/tmp/test_hash.bin", "wb");
        fwrite(data, 1, strlen(data), f);
        fclose(f);
    }
    auto result = cv.compute("/tmp/test_hash.bin", ChecksumType::SHA256);
    assert(result.verified);
    assert(strlen(result.hash) == 64); // SHA256 = 64 hex chars
    PASS();

    TEST("MD5 compute");
    auto md5 = cv.compute("/tmp/test_hash.bin", ChecksumType::MD5);
    assert(md5.verified);
    assert(strlen(md5.hash) == 32); // MD5 = 32 hex chars
    PASS();

    TEST("File comparison");
    {
        FILE* f = fopen("/tmp/test_hash_copy.bin", "wb");
        fwrite(data, 1, strlen(data), f);
        fclose(f);
    }
    assert(ChecksumVerifier::files_identical("/tmp/test_hash.bin", "/tmp/test_hash_copy.bin"));
    PASS();
}

void test_logger() {
    TEST("Log entries");
    Logger logger("/tmp/test_recovery.log");
    logger.log(Operation::BACKUP_SYSTEM, Status::COMPLETED, "Test backup completed");
    logger.log(Operation::INSTALL_SYSTEM, Status::FAILED, "Test install failed: no space");
    assert(logger.entry_count() >= 2);
    PASS();

    TEST("Get entries by operation");
    auto entries = logger.get_entries_by_operation(Operation::BACKUP_SYSTEM, 5);
    assert(!entries.empty());
    PASS();
}

void test_backup_engine() {
    TEST("Backup system partition");
    PartitionManager pm;
    ChecksumVerifier cv;
    BackupEngine engine(&pm, &cv);

    pm.create_simulated_partition("system", "/tmp/test_system.img",
                                   64*1024*1024, "ext4");
    // Write test data
    system("dd if=/dev/urandom of=/tmp/test_system.img bs=1M count=16 2>/dev/null");

    bool ok = engine.backup_system("system", "/tmp/test_backup.img.xz", nullptr);
    assert(ok);
    PASS();

    TEST("Verify backup file");
    auto checksum = cv.compute("/tmp/test_backup.img.xz", ChecksumType::SHA256);
    assert(checksum.verified);
    PASS();

    // Cleanup
    std::remove("/tmp/test_system.img");
    std::remove("/tmp/test_backup.img.xz");
}

int main() {
    std::cout << "\n═══════════════════════════════════════════════\n";
    std::cout << "  RK3588 Recovery System — Unit Tests v2.0\n";
    std::cout << "═══════════════════════════════════════════════\n\n";

    test_partition_manager();
    test_checksum_verifier();
    test_logger();
    test_backup_engine();

    std::cout << "\n═══════════════════════════════════════════════\n";
    std::cout << "  Results: " << tests_passed << " passed, "
              << tests_failed << " failed\n";
    std::cout << "═══════════════════════════════════════════════\n";

    // Cleanup temp files
    std::remove("/tmp/test_hash.bin");
    std::remove("/tmp/test_hash_copy.bin");
    std::remove("/tmp/test_system.img");
    std::remove("/tmp/test_backup.img.xz");

    return tests_failed > 0 ? 1 : 0;
}
