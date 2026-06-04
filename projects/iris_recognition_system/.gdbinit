# =============================================================================
# GDB initialization file for Iris Recognition System
#
# Usage:
#   gdb -x .gdbinit ./build/iris_app
#   or GDB will auto-load this file if ~/.gdbinit contains:
#       set auto-load safe-path /home/sa/Lab/Programming/aspira2026
# =============================================================================

# ── History ──────────────────────────────────────────────────────────────────
set history save on
set history filename .gdb_history
set history size 10000

# ── Pagination ───────────────────────────────────────────────────────────────
set pagination off

# ── Print settings ───────────────────────────────────────────────────────────
set print pretty on
set print array on
set print array-indexes on
set print object on
set print static-members on
set print vtbl on
set print demangle on

# ── Backtrace ────────────────────────────────────────────────────────────────
set print frame-arguments all

# ── Disassembly flavor (Intel syntax is easier to read) ──────────────────────
set disassembly-flavor intel

# ── Confirmations ────────────────────────────────────────────────────────────
set confirm off

# ── Remote debugging ─────────────────────────────────────────────────────────
# set architecture i386:x86-64:intel

# ── Log to file ──────────────────────────────────────────────────────────────
# set logging file gdb_output.log
# set logging on

# =============================================================================
# Custom commands
# =============================================================================

# Print iris namespace types
define piris
    echo Usage: piris <template_address>\n
    echo Prints IrisTemplate fields: iris_code (first 32 bytes), quality\n
    set $t = ($arg0)
    printf "quality: %.4f\n", $t->quality_score
    printf "mask[0..7]: "
    print $t->mask_code[0]@8
end

# Print cv::Mat basic info
define pmat
    set $m = ($arg0)
    printf "Mat: rows=%d, cols=%d, channels=%d, dims=%d, elemSize=%lu, type=%d\n", \
        $m->rows, $m->cols, $m->channels(), $m->dims, $m->elemSize(), $m->type()
end

# Print cv::Mat with data (uchar preview, first few elements)
define pmatdata
    pmat $arg0
    set $m = ($arg0)
    if $m->dims <= 2
        if $m->type() == 0 || $m->type() == 5 || $m->type() == 6
            printf "data[0..9]: "
            print ((float*)$m->data)[0]@10
        else
            printf "data[0..9]: "
            print ((unsigned char*)$m->data)[0]@10
        end
    end
end

echo ===== Iris Recognition System GDB Config Loaded =====\n
