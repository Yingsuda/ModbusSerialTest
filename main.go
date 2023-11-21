package main

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"gitee.com/iotprotocol/modbus/modbusUtil"
	"gitee.com/iotprotocol/modbus/protocol/Das/PollDas"
	"gitee.com/iotprotocol/tools/logger"
	"go.bug.st/serial"
)

type SerialAddr struct {
	network string
	//串口ID
	id string
	//波特率
	baudRate int
	//停止位
	stopBits byte
	//校验位
	parity string
	//数据位
	dataBits byte
}

// Network :network type :"serial"
func (a *SerialAddr) Network() string {
	return a.network
}

//String: string format of serial address,format : baud rate,data bits, parity, stop bits
//for example:"9600,8,NONE,1"
func (a *SerialAddr) String() string {
	return fmt.Sprintf("%v,%v,%v,%v", a.baudRate, a.dataBits, a.parity, a.stopBits)
}

type Serial struct {
	addr     *SerialAddr
	conn     serial.Port
	PortName string
	Baud     int
	DataBits int
	StopBits int
	Parity   string
}

func (s *Serial) LocalAddr() net.Addr {
	return s.addr
}

func (s *Serial) RemoteAddr() net.Addr {
	return s.addr
}

// SetDeadline set read deadline,ignore write deadline
func (s *Serial) SetDeadline(t time.Time) error {
	return s.SetReadDeadline(t)
}

// SetReadDeadline set write timeout, set no timeout if t is current time or t less current time
func (s *Serial) SetReadDeadline(t time.Time) error {
	timeout := t.UnixNano() - time.Now().UnixNano()
	if timeout <= 0 {
		return s.conn.SetReadTimeout(serial.NoTimeout)
	}
	return s.conn.SetReadTimeout(time.Duration(timeout))
}

// SetWriteDeadline write deadline ignored
func (s *Serial) SetWriteDeadline(t time.Time) error {
	return nil
}

func (s *Serial) Open() (net.Conn, error) {
	if s.conn != nil {
		return s, nil
	}
	mode := &serial.Mode{}
	mode.BaudRate = s.Baud
	mode.DataBits = s.DataBits
	switch s.StopBits {
	case 1:
		mode.StopBits = serial.OneStopBit
	case 2:
		mode.StopBits = serial.TwoStopBits
	default:
		return nil, fmt.Errorf("unknown stop bit '%v'", s.StopBits)
	}
	switch strings.ToUpper(s.Parity) {
	case "ODD":
		mode.Parity = serial.OddParity
	case "EVEN":
		mode.Parity = serial.EvenParity
	case "NONE":
		mode.Parity = serial.NoParity
	case "MARK":
		mode.Parity = serial.MarkParity
	case "SPACE":
		mode.Parity = serial.SpaceParity
	default:
		return nil, fmt.Errorf("unknown check bit '%v'", s.Parity)
	}

	open, err := serial.Open(s.PortName, &serial.Mode{})
	if err != nil {
		return nil, err
	}
	err = open.SetMode(mode)
	if err != nil {
		return nil, err
	}
	//err = open.SetReadTimeout(time.Duration(s.option.readTimeout) * time.Millisecond)
	//if err != nil {
	//	dlog.Warn("Set serial read timeout error:", err)
	//}
	s.conn = open
	return s, nil
}

func (s *Serial) Write(buf []byte) (n int, err error) {
	return s.conn.Write(buf)
}

func (s *Serial) Read(buf []byte) (num int, err error) {
	num, err = s.conn.Read(buf)
	if err != nil {
		return
	}
	if num == 0 {
		return 0, errors.New("i/o timeout")
	}
	return
}

func (s *Serial) Close() error {
	err := s.conn.Close()
	if err != nil {
		return err
	}
	s.conn = nil
	return nil
}

func main() {
	s := Serial{
		PortName: "COM5",
		Baud:     9600,
		DataBits: 8,
		StopBits: 1,
		Parity:   "NONE",
	}

	conn, err := s.Open()
	if err != nil {
		fmt.Println("serial open err:", err)
		return
	}

	start := time.Now()
	conn.SetDeadline(time.Now().Add(time.Duration(1000)*time.Millisecond - time.Since(start)))

	handle := PollDas.NewRTUPollHandler().WithSlaveId(1).WithTimeout(5000).WithConnection(conn)
	client := PollDas.NewClient(handle, logger.NewDefaultLogger())

	res, err := client.ReadHoldingRegisters(0, 2)
	if err != nil {
		fmt.Println("modbus read HoldRegister err:", err)
		return
	}
	fmt.Printf("Data:% 02X\n", res)

	fmt.Println("Control =================>>>")

	data, err := modbusUtil.MakeControlDataCommand("INT16", float64(223))
	if err != nil {
		fmt.Println("modbus MakeControlCommand err:", err)
		return
	}

	_, err = client.WriteSingleRegister(4, data)
	if err != nil {
		fmt.Println("WriteSingleRegister err:", err)
		return
	}
}
