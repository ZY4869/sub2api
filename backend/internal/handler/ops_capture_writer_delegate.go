package handler

import (
	"bufio"
	"errors"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

var errOpsCaptureWriterReleased = errors.New("ops capture writer delegate released")

func opsWriterHeader(rw gin.ResponseWriter) http.Header {
	if rw == nil {
		return http.Header{}
	}
	return rw.Header()
}

func opsWriterStatus(rw gin.ResponseWriter) int {
	if rw == nil {
		return http.StatusOK
	}
	return rw.Status()
}

func opsWriterSize(rw gin.ResponseWriter) int {
	if rw == nil {
		return 0
	}
	return rw.Size()
}

func opsWriterWritten(rw gin.ResponseWriter) bool {
	return rw != nil && rw.Written()
}

func opsWriterWriteHeaderNow(rw gin.ResponseWriter) {
	if rw != nil {
		rw.WriteHeaderNow()
	}
}

func opsWriterWriteHeader(rw gin.ResponseWriter, code int) {
	if rw != nil {
		rw.WriteHeader(code)
	}
}

func opsWriterWrite(rw gin.ResponseWriter, data []byte) (int, error) {
	if rw == nil {
		return 0, errOpsCaptureWriterReleased
	}
	return rw.Write(data)
}

func opsWriterWriteString(rw gin.ResponseWriter, value string) (int, error) {
	if rw == nil {
		return 0, errOpsCaptureWriterReleased
	}
	return rw.WriteString(value)
}

func opsWriterFlush(rw gin.ResponseWriter) {
	if rw != nil {
		rw.Flush()
	}
}

func opsWriterHijack(rw gin.ResponseWriter) (net.Conn, *bufio.ReadWriter, error) {
	if rw == nil {
		return nil, nil, errOpsCaptureWriterReleased
	}
	return rw.Hijack()
}

func opsWriterCloseNotify(rw gin.ResponseWriter) <-chan bool {
	if rw == nil {
		ch := make(chan bool)
		close(ch)
		return ch
	}
	return rw.CloseNotify()
}

func opsWriterPusher(rw gin.ResponseWriter) http.Pusher {
	if rw == nil {
		return nil
	}
	return rw.Pusher()
}
