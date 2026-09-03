// Package proxyutil 提供统一的代理配置功能
//
// 支持的代理协议：
//   - HTTP/HTTPS: 通过 Transport.Proxy 设置
//   - SOCKS5: 通过 Transport.DialContext 设置（客户端本地解析 DNS）
//   - SOCKS5H: 通过 Transport.DialContext 设置（代理端远程解析 DNS，推荐）
//
// 注意：proxyurl.Parse() 会自动将 socks5:// 升级为 socks5h://，
// 确保 DNS 也由代理端解析，防止 DNS 泄漏。
package proxyutil

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"golang.org/x/net/proxy"
)

const fallbackProxyDialTimeout = 10 * time.Second

// ConfigureTransportProxy 根据代理 URL 配置 Transport
//
// 支持的协议：
//   - http/https: 设置 transport.Proxy
//   - socks5: 设置 transport.DialContext（客户端本地解析 DNS）
//   - socks5h: 设置 transport.DialContext（代理端远程解析 DNS，推荐）
//
// 参数：
//   - transport: 需要配置的 http.Transport
//   - proxyURL: 代理地址，nil 表示直连
//
// 返回：
//   - error: 代理配置错误（协议不支持或 dialer 创建失败）
func ConfigureTransportProxy(transport *http.Transport, proxyURL *url.URL) error {
	return configureTransportProxy(transport, proxyURL, nil)
}

// ConfigureTransportProxyWithDialer 根据代理 URL 配置 Transport，并允许调用方
// 指定连接代理服务器本身时使用的底层 dialer。
func ConfigureTransportProxyWithDialer(transport *http.Transport, proxyURL *url.URL, forward proxy.Dialer) error {
	if forward == nil {
		forward = proxy.Direct
	}
	return configureTransportProxy(transport, proxyURL, forward)
}

func configureTransportProxy(transport *http.Transport, proxyURL *url.URL, forward proxy.Dialer) error {
	if proxyURL == nil {
		return nil
	}

	scheme := strings.ToLower(proxyURL.Scheme)
	switch scheme {
	case "http", "https":
		if forward != nil {
			transport.DialContext = dialContextFromProxyDialer(forward)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		return nil

	case "socks5", "socks5h":
		if forward == nil {
			forward = proxy.Direct
		}
		dialer, err := proxy.FromURL(proxyURL, forward)
		if err != nil {
			return fmt.Errorf("create socks5 dialer: %w", err)
		}
		// 优先使用支持 context 的 DialContext，以支持请求取消和超时
		if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
			transport.DialContext = contextDialer.DialContext
		} else {
			transport.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				return dialWithContextDeadline(ctx, dialer, network, addr)
			}
		}
		return nil

	default:
		return fmt.Errorf("unsupported proxy scheme: %s", scheme)
	}
}

func dialContextFromProxyDialer(dialer proxy.Dialer) func(context.Context, string, string) (net.Conn, error) {
	if contextDialer, ok := dialer.(proxy.ContextDialer); ok {
		return contextDialer.DialContext
	}
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialWithContextDeadline(ctx, dialer, network, addr)
	}
}

func dialWithContextDeadline(ctx context.Context, dialer proxy.Dialer, network, addr string) (net.Conn, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, fallbackProxyDialTimeout)
		defer cancel()
	}

	type result struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		conn, err := dialer.Dial(network, addr)
		select {
		case resultCh <- result{conn: conn, err: err}:
		case <-ctx.Done():
			if conn != nil {
				_ = conn.Close()
			}
		}
	}()

	select {
	case result := <-resultCh:
		return result.conn, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
