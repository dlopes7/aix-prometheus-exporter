package c_api

/*
#cgo LDFLAGS: -lperfstat
#include <stdlib.h>
#include <libperfstat.h>
#include <string.h>
#include <time.h>

int getCPUTicks(u_longlong_t **cputicks, size_t *cpu_ticks_len) {
	int i, ncpus, cputotal;
	perfstat_id_t firstcpu;
	perfstat_cpu_t *statp;

	cputotal =  perfstat_cpu(NULL, NULL, sizeof(perfstat_cpu_t), 0);
	if (cputotal <= 0){
        return -1;
   }

	statp = calloc(cputotal, sizeof(perfstat_cpu_t));
	if(statp==NULL){
			return -1;
	}
	ncpus = perfstat_cpu(&firstcpu, statp, sizeof(perfstat_cpu_t), cputotal);
	*cpu_ticks_len = ncpus*4;

	u_longlong_t user, wait, sys, idle;
	*cputicks = (u_longlong_t *) malloc(sizeof(u_longlong_t)*(*cpu_ticks_len));
	for (i = 0; i < ncpus; i++) {
		int offset = 4 * i;
		(*cputicks)[offset] = statp[i].user;
		(*cputicks)[offset+1] = statp[i].sys;
		(*cputicks)[offset+2] = statp[i].wait;
		(*cputicks)[offset+3] = statp[i].idle;
	}
	return 0;
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

const ClocksPerSec = float64(C.CLK_TCK)
const maxCPUTimesLen = 1024 * 4

func GetAIXCPUTimes() ([]float64, error) {

	var (
		cpuTimesC      *C.u_longlong_t
		cpuTimesLength C.size_t
	)

	if C.getCPUTicks(&cpuTimesC, &cpuTimesLength) == -1 {
		return nil, errors.New("could not retrieve CPU times")
	}
	defer C.free(unsafe.Pointer(cpuTimesC))

	cput := (*[maxCPUTimesLen]C.u_longlong_t)(unsafe.Pointer(cpuTimesC))[:cpuTimesLength:cpuTimesLength]

	cpuTicks := make([]float64, cpuTimesLength)
	for i, value := range cput {
		cpuTicks[i] = float64(value) / ClocksPerSec
	}
	return cpuTicks, nil

}
