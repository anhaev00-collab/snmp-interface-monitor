package main

import (
	"flag"
	"fmt"
	"log"
	"math/big"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"
)
const (

	oidIfName        = ".1.3.6.1.2.1.31.1.1.1.1"

	oidIfOperStatus  = ".1.3.6.1.2.1.2.2.1.8"

	oidIfHCInOctets  = ".1.3.6.1.2.1.31.1.1.1.6"

	oidIfHCOutOctets = ".1.3.6.1.2.1.31.1.1.1.10"

)

type Iface struct {

Index     int

	Name      string

	Status    int

InOctets  uint64

OutOctets uint64

}


func main() {

	host := flag.String("host", "127.0.0.1", "SNMP host")

	community := flag.String("community", "public", "SNMP community")

	interval := flag.Int("interval", 1, "Update interval in seconds")

	flag.Parse()



	snmp := &gosnmp.GoSNMP{

		Target:    *host,

		Port:      161,

		Community: *community,

		Version:   gosnmp.Version2c,

		Timeout:   2 * time.Second,

		Retries:   1,

}



	err := snmp.Connect()

	if err != nil {

		log.Fatalf("SNMP connect error: %v", err)

}

	defer snmp.Conn.Close()



	previous := make(map[int]Iface)

	for {
		current, err := readInterfaces(snmp)
		if err != nil {
			log.Printf("SNMP read error: %v", err)
			time.Sleep(time.Duration(*interval) * time.Second)
			continue
		}

		clearScreen()
		printTable(current, previous, *interval)

		previous = current
		time.Sleep(time.Duration(*interval) * time.Second)
	}
}

func readInterfaces(snmp *gosnmp.GoSNMP) (map[int]Iface, error) {
	result := make(map[int]Iface)

	err := walk(snmp, oidIfName, func(index int, pdu gosnmp.SnmpPDU) {
		iface := result[index]
		iface.Index = index
		iface.Name = toString(pdu.Value)
		result[index] = iface
})
	if err != nil {
		return nil, err
	}

	err = walk(snmp, oidIfOperStatus, func(index int, pdu gosnmp.SnmpPDU) {
		iface := result[index]
		iface.Index = index
		iface.Status = int(toUint64(pdu.Value))
		result[index] = iface
})
	if err != nil {
		return nil, err
}

	err = walk(snmp, oidIfHCInOctets, func(index int, pdu gosnmp.SnmpPDU) {
		iface := result[index]
		iface.Index = index
		iface.InOctets = toUint64(pdu.Value)
		result[index] = iface
	})
	if err != nil {
		return nil, err
}

	err = walk(snmp, oidIfHCOutOctets, func(index int, pdu gosnmp.SnmpPDU) {
		iface := result[index]
		iface.Index = index
		iface.OutOctets = toUint64(pdu.Value)
		result[index] = iface
})
	if err != nil {
		return nil, err
	}

	return result, nil
}

func walk(snmp *gosnmp.GoSNMP, baseOID string, handler func(index int, pdu gosnmp.SnmpPDU)) error {
	return snmp.Walk(baseOID, func(pdu gosnmp.SnmpPDU) error {
		index, err := getIndex(pdu.Name, baseOID)
		if err != nil {
			return nil
		}

		handler(index, pdu)
		return nil
})
}

func getIndex(fullOID string, baseOID string) (int, error) {
	fullOID = strings.TrimPrefix(fullOID, ".")
	baseOID = strings.TrimPrefix(baseOID, ".")

	suffix := strings.TrimPrefix(fullOID, baseOID)
	suffix = strings.TrimPrefix(suffix, ".")

	return strconv.Atoi(suffix)
}

func toString(value interface{}) string {
	switch v := value.(type) {
	case []byte:
		return string(v)
	case string:
		return v
	default:
		return fmt.Sprintf("%v", v)
}
}

func toUint64(value interface{}) uint64 {
	switch v := value.(type) {
	case uint:
		return uint64(v)
	case uint32:
		return uint64(v)
	case uint64:
		return v
	case int:
		return uint64(v)
	case int32:
		return uint64(v)
	case int64:
		return uint64(v)
	case *big.Int:
		return v.Uint64()
	default:
		return 0
}
}

func statusText(status int) string {
	switch status {
	case 1:
		return "up"
	case 2:
		return "down"
	case 3:
		return "testing"
	default:
		return "unknown"
	}
}

func printTable(current map[int]Iface, previous map[int]Iface, interval int) {
	indexes := make([]int, 0, len(current))

	for index := range current {
		indexes = append(indexes, index)
}

	sort.Ints(indexes)

	fmt.Printf("%-5s %-15s %-10s %-15s %-15s %-12s %-12s\n",
"IDX", "INTERFACE", "STATUS", "IN_MB", "OUT_MB", "IN_Mbps", "OUT_Mbps")
	fmt.Println(strings.Repeat("-", 90))

	for _, index := range indexes {
		iface := current[index]

		var inMbps float64
		var outMbps float64

		if prev, ok := previous[index]; ok {
			if iface.InOctets >= prev.InOctets {
				diff := iface.InOctets - prev.InOctets
				inMbps = float64(diff*8) / float64(interval) / 1000000
			}

			if iface.OutOctets >= prev.OutOctets {
				diff := iface.OutOctets - prev.OutOctets
				outMbps = float64(diff*8) / float64(interval) / 1000000
}
		}

		fmt.Printf("%-5d %-15s %-10s %-15.2f %-15.2f %-12.2f %-12.2f\n",
			iface.Index,
			iface.Name,
			statusText(iface.Status),
			float64(iface.InOctets)/1024/1024,
			float64(iface.OutOctets)/1024/1024,
			inMbps,
			outMbps,
)
	}
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}
