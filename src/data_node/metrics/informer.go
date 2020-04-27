package main

import (
	"exec"
	"fmt"
)

type Metric struct {
	Name    string
	Data    float64
	Expired int
	Src     string
}

func CPUInfo() float64 {
	cmdline := exec.Command("/bin/cat", "/proc/cpuinfo")
	if outputs, err := cmdline.Output(); err != nil {
		log.Fatal("command error", err)
	} else {
		cores := strings.Count(string(outputs), "processor")
		if cores <= 0 {
			log.Fatal("Wrong Core numbers")
		}
		return cores
	}
	return 0
}

func MemoryInfo() float64 {
	if fd, err := os.Open("/proc/meminfo"); err != nil {
		log.Fatal(err)
	}
	scanner := bufio.NewScanner(fd)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "MemFree") {
			words := strings.Fields(line)
			if len(words) < 3 {
				log.Fatal("Wrong Format of meminfo")
			} else {
				log.Println(words[0], words[1], words[2])
				freemem, err := strconv.Atoi(words[2])
				if err != nil {
					log.Fatal(err)
				}
				return float64(freemem)
			}
		}
	}
	return -1.0
}

func ChurnInfo() float64 {
	cmd := exec.Command("/bin/last", "-x")
	if outputs, err := cmd.Output(); err != nil {
		log.Fatal(err)
	} else {
		churn := strings.Count(string(outputs), "shutdown")
		if churn > 0 {
			return float64(churn)
		}
	}
	return 0
}

func getIPv4() string {
	ifaces, _ := net.Interfaces()
	// handle err
	for _, i := range ifaces {
		if strings.Contains(i.Name, "eth") {
			addrs, _ := i.Addrs()
			// handle err
			for _, addr := range addrs {
				var ip net.IP
				//type Addr {
				//    Network() string
				//    String()  string
				//}
				switch v := addr.(type) {
				case *net.IPNet:
					ip = v.IP
				case *net.IPAddr:
					ip = v.IP
				}
				// process IP address
				if ip.To4() != nil {
					return ip.String()
				}
			}
		}
	}
	return ""
}

func PostJson(mp *Metric, host string) {
	jsonBytes, err := json.Marshal(m)
	if err != nil {
		log.Fatal(err)
	}
	url := "http://" + host + ":28001" + "/acceptmetrics"
	body := bytes.NewReader(jsonBytes)
	http.Post()
}

func SendMetrics() {
	for {
		addr := getIPv4()
		log.Printf("%s Start to Send Metrics", addr)
		m := Metric{
			Name: "cpu",
			Data: CPUInfo(),
			Src:  addr,
		}

		time.Sleep(30 * time.Seconds)
	}
}

func main() {
	fmt.Println("vim-go")
}
