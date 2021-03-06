package ipc

import (
	"fmt"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/req"
	"github.com/go-mangos/mangos/protocol/sub"
	//"github.com/go-mangos/mangos/transport/ipc"
	"github.com/go-mangos/mangos/transport/tcp"

	"github.com/cloudimmunity/docker-slim/messages"
)

func InitContainerChannels(dockerHostIp, cmdChannelPort, evtChannelPort string) error {
	cmdChannelAddr = fmt.Sprintf("tcp://%v:%v", dockerHostIp, cmdChannelPort)
	evtChannelAddr = fmt.Sprintf("tcp://%v:%v", dockerHostIp, evtChannelPort)
	log.Debugf("cmdChannelAddr=%v evtChannelAddr=%v\n", cmdChannelAddr, evtChannelAddr)

	//evtChannelAddr = fmt.Sprintf("ipc://%v/ipc/docker-slim-sensor.events.ipc", localVolumePath)
	//cmdChannelAddr = fmt.Sprintf("ipc://%v/ipc/docker-slim-sensor.cmds.ipc", localVolumePath)

	var err error
	evtChannel, err = newEvtChannel(evtChannelAddr)
	if err != nil {
		return err
	}
	cmdChannel, err = newCmdClient(cmdChannelAddr)
	if err != nil {
		return err
	}

	return nil
}

func SendContainerCmd(cmd messages.Message) (string, error) {
	return sendCmd(cmdChannel, cmd)
}

func GetContainerEvt() (string, error) {
	return getEvt(evtChannel)
}

func ShutdownContainerChannels() {
	shutdownEvtChannel()
	shutdownCmdChannel()
}

//var cmdChannelAddr = "ipc:///tmp/docker-slim-sensor.cmds.ipc"
var cmdChannelAddr = "tcp://127.0.0.1:65501"
var cmdChannel mangos.Socket

func newCmdClient(addr string) (mangos.Socket, error) {
	socket, err := req.NewSocket()
	if err != nil {
		return nil, err
	}

	if err := socket.SetOption(mangos.OptionSendDeadline, time.Second*3); err != nil {
		socket.Close()
		return nil, err
	}

	if err := socket.SetOption(mangos.OptionRecvDeadline, time.Second*3); err != nil {
		socket.Close()
		return nil, err
	}

	//socket.AddTransport(ipc.NewTransport())
	socket.AddTransport(tcp.NewTransport())
	if err := socket.Dial(addr); err != nil {
		socket.Close()
		return nil, err
	}

	return socket, nil
}

func shutdownCmdChannel() {
	if cmdChannel != nil {
		cmdChannel.Close()
		cmdChannel = nil
	}
}

func sendCmd(channel mangos.Socket, cmd messages.Message) (string, error) {
	sendTimeouts := 0
	recvTimeouts := 0

	log.Debugf("sendCmd(%s)\n", cmd)
	for {
		sendData, err := messages.Encode(cmd)
		if err != nil {
			log.Info("sendCmd(): malformed cmd - ", err)
			return "", err
		}

		if err := channel.Send(sendData); err != nil {
			switch err {
			case mangos.ErrSendTimeout:
				log.Info("sendCmd(): send timeout...")
				sendTimeouts++
				if sendTimeouts > 3 {
					return "", err
				}
			default:
				return "", err
			}
		}

		response, err := channel.Recv()
		if err != nil {
			switch err {
			case mangos.ErrRecvTimeout:
				log.Info("sendCmd(): receive timeout...")
				recvTimeouts++
				if recvTimeouts > 3 {
					return "", err
				}
			default:
				return "", err
			}
		}

		return string(response), nil
	}
}

var evtChannelAddr = "tcp://127.0.0.1:65502"

//var evtChannelAddr = "ipc:///tmp/docker-slim-sensor.events.ipc"
var evtChannel mangos.Socket

func newEvtChannel(addr string) (mangos.Socket, error) {
	socket, err := sub.NewSocket()
	if err != nil {
		return nil, err
	}

	if err := socket.SetOption(mangos.OptionRecvDeadline, time.Second*120); err != nil {
		socket.Close()
		return nil, err
	}

	//socket.AddTransport(ipc.NewTransport())
	socket.AddTransport(tcp.NewTransport())
	if err := socket.Dial(addr); err != nil {
		socket.Close()
		return nil, err
	}

	err = socket.SetOption(mangos.OptionSubscribe, []byte(""))
	if err != nil {
		return nil, err
	}

	return socket, nil
}

func shutdownEvtChannel() {
	if evtChannel != nil {
		evtChannel.Close()
		evtChannel = nil
	}
}

func getEvt(channel mangos.Socket) (string, error) {
	log.Debug("getEvt()")
	evt, err := channel.Recv()
	if err != nil {
		return "", err
	}

	return string(evt), nil
}
