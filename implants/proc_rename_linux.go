//go:build linux

package main

/*
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <sys/prctl.h>
#include <time.h>


#define TASK_COMM_LEN 16

void setProcNameNative(const char* newname_orig) {
    char newname_truncated[TASK_COMM_LEN];
    strncpy(newname_truncated, newname_orig, TASK_COMM_LEN - 1);
    newname_truncated[TASK_COMM_LEN - 1] = '\0';
    prctl(PR_SET_NAME, newname_truncated, 0, 0, 0);
}

void seedRand() {
	srand((unsigned int)time(NULL));
}
*/
import "C"
import (
	"unsafe"
)

func init() {
	C.seedRand()

	randomizedPrctlName := generateLegitLookingName()
	csPrctlName := C.CString(randomizedPrctlName)
	defer C.free(unsafe.Pointer(csPrctlName))

	C.setProcNameNative(csPrctlName)

}

func generateLegitLookingName() string {

	names := []string{
		"kthreadd",
		"rcu_sched",
		"ksoftirqd/0",
		"kworker/u64:0",
		"migration/0",
		"watchdog/0",
		"events/0",
		"dbus-daemon",
		"systemd-resolve",
		"gvfsd-fuse",
		"anacron",
	}
	return names[int(C.rand())%len(names)]
}

/*
this was supposed to overwrite argv but turns out it's pretty tricky
keeping it here in case i want to revisit later
func overwriteArgv(newName string) {
	args := os.Args
	if len(args) == 0 {
		return
	}
	argv0 := []byte(args[0])
	newNameBytes := []byte(newName)
	copy(argv0, newNameBytes)
	if len(newNameBytes) < len(argv0) {
		for i := len(newNameBytes); i < len(argv0); i++ {
			argv0[i] = 0
		}
	}

	// yeah this doesn't work as expected because go manages argv differently
	// would need to go deeper into runtime stuff which is overkill for now

}
*/
