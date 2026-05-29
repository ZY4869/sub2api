package repository

import (
	"io"
	"sync"
)

type trackedBody struct {
	io.ReadCloser // 原始响应体
	once          sync.Once
	onClose       func() // 关闭时的回调函数
}

// Close 关闭响应体并执行回调
// 使用 sync.Once 确保回调只执行一次

func (b *trackedBody) Close() error {
	err := b.ReadCloser.Close()
	if b.onClose != nil {
		b.once.Do(b.onClose)
	}
	return err
}

// wrapTrackedBody 包装响应体以跟踪关闭事件
// 用于在响应体关闭时更新 inFlight 计数
//
// 参数:
//   - body: 原始响应体
//   - onClose: 关闭时的回调函数
//
// 返回:
//   - io.ReadCloser: 包装后的响应体

func wrapTrackedBody(body io.ReadCloser, onClose func()) io.ReadCloser {
	if body == nil {
		return body
	}
	return &trackedBody{ReadCloser: body, onClose: onClose}
}
