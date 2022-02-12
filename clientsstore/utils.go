package clientsstore

import "chromebalancer/utils"

func (cs *ChromesStore) GenIdlePort() int {
	p := 0

	for {
		p = utils.RandInt(PortIntervalStart, PortIntervalEnd)

		if !cs.busyPorts[p] {
			break
		}
	}

	return p
}
