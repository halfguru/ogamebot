package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	patchMain()
	patchLogin()
}

func patchMain() {
	path := "/go/src/ogame/cmd/ogamed/main.go"
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading main.go: %v\n", err)
		os.Exit(1)
	}
	content := string(data)

	if strings.Contains(content, "OGAMED_BEARER_TOKEN") {
		fmt.Println("main.go already patched")
		return
	}

	flagInsert := `&cli.StringFlag{
				Name:     "bearer-token",
				Sources:  cli.EnvVars("OGAMED_BEARER_TOKEN"),
				Usage:    "Gameforge bearer token (gf-token-production)",
			},`
	content = strings.Replace(content,
		`&cli.StringFlag{
				Name:     "proxy",
				Sources:  cli.EnvVars("OGAMED_PROXY"),`,
		flagInsert+"\n"+`&cli.StringFlag{
				Name:     "proxy",
				Sources:  cli.EnvVars("OGAMED_PROXY"),`,
		1,
	)

	content = strings.Replace(content,
		"Proxy:          proxyAddr,",
		"Proxy:          proxyAddr,\n\t\tBearerToken:    c.String(\"bearer-token\"),",
		1,
	)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "writing main.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Patched main.go with OGAMED_BEARER_TOKEN support")
}

func patchLogin() {
	path := "/go/src/ogame/pkg/wrapper/ogame_login.go"
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "reading ogame_login.go: %v\n", err)
		os.Exit(1)
	}
	content := string(data)

	if strings.Contains(content, "DEBUG_PATCH") {
		fmt.Println("ogame_login.go already patched")
		return
	}

	debugLog := `log.Printf("DEBUG_PATCH wrapLoginWithExistingCookies: token=%q phpSessID=%q", token, phpSessID)`
	content = strings.Replace(content,
		"useCookies, usePhpSessID, err = b.loginWithBearerToken(token, phpSessID)",
		debugLog+"\n\t\t"+"useCookies, usePhpSessID, err = b.loginWithBearerToken(token, phpSessID)",
		1,
	)

	debugLog2 := `log.Printf("DEBUG_PATCH loginWithBearerToken: token=%q phpSessID=%q", token, phpSessID)`
	content = strings.Replace(content,
		"var didPart1n2 bool",
		debugLog2+"\n\tvar didPart1n2 bool",
		1,
	)

	debugLog3 := `log.Printf("DEBUG_PATCH loginPart1 err=%v", err)`
	content = strings.Replace(content,
		"} else {\n\t\t\ttoken = \"\"",
		"} else {\n\t\t\t"+debugLog3+"\n\t\t\ttoken = \"\"",
		1,
	)

	debugLog4 := `log.Printf("DEBUG_PATCH fast path failed, falling through to beginning")`
	content = strings.Replace(content,
		"beginning:",
		debugLog4+"\nbeginning:",
		1,
	)

	content = strings.Replace(content,
		`"context"`,
		`"context"\n\t"log"`,
		1,
	)

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "writing ogame_login.go: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Patched ogame_login.go with debug logging")
}
