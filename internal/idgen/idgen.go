// Package idgen 生成短链使用的 base62 短码。
package idgen

import (
	"crypto/rand"
	"errors"
	"math/big"
)

const alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

// ErrTooShort 表示请求的长度不足以支撑碰撞重试。
var ErrTooShort = errors.New("idgen: code length too short")

// NewCode 返回长度为 n 的随机 base62 短码。
func NewCode(n int) (string, error) {
	if n <= 0 {
		return "", ErrTooShort
	}
	out := make([]byte, n)
	max := big.NewInt(int64(len(alphabet)))
	for i := range out {
		idx, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		out[i] = alphabet[idx.Int64()]
	}
	return string(out), nil
}

// UniqueCode 在 collide 返回 true 时持续重试，最多尝试 maxTries 次。
// collide 用于判断生成的短码是否已存在。
func UniqueCode(n, maxTries int, collide func(string) bool) (string, error) {
	if n <= 0 {
		return "", ErrTooShort
	}
	if maxTries <= 0 {
		maxTries = 8
	}
	for i := 0; i < maxTries; i++ {
		c, err := NewCode(n)
		if err != nil {
			return "", err
		}
		if !collide(c) {
			return c, nil
		}
	}
	return "", errors.New("idgen: failed to generate unique code after retries")
}
