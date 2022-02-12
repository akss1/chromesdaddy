package clientsstore

import "chromesdaddy/utils"

func (cs *ClientsStore) GenIdlePort() int {
	p := 0

	for {
		p = utils.RandInt(PortIntervalStart, PortIntervalEnd)

		if !cs.busyPorts[p] {
			break
		}
	}

	return p
}
