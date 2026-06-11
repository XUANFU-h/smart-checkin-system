package main

import (
	"encoding/hex"
	"fmt"

	"github.com/tjfoc/gmsm/sm2"
	"github.com/tjfoc/gmsm/x509"
)

func main() {
	priv, err := sm2.GenerateKey(nil)
	if err != nil {
		fmt.Println("生成密钥失败：", err)
		return
	}

	// v1.4.1 的真实序列化在 x509 模块下
	privBytes, _ := x509.MarshalSm2UnecryptedPrivateKey(priv)
	pubBytes, _ := x509.MarshalSm2PublicKey(&priv.PublicKey)

	fmt.Println("=")
	fmt.Println("✅ 后端SM2私钥：")
	fmt.Println(hex.EncodeToString(privBytes))
	fmt.Println("=")
	fmt.Println("✅ 前端SM2公钥：")
	fmt.Println(hex.EncodeToString(pubBytes))
	fmt.Println("=")
}
