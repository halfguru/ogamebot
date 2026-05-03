package ogamed

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"testing"
)

func TestCaptchaTextPairsNotEmpty(t *testing.T) {
	if len(captchaTextPairs) == 0 {
		t.Fatal("captchaTextPairs should not be empty")
	}
	if len(captchaTextPairs) < 1000 {
		t.Fatalf("captchaTextPairs should have at least 1000 entries, got %d", len(captchaTextPairs))
	}
}

func TestCaptchaImagePairsNotEmpty(t *testing.T) {
	if len(captchaImagePairs) == 0 {
		t.Fatal("captchaImagePairs should not be empty")
	}
	if len(captchaImagePairs) < 50 {
		t.Fatalf("captchaImagePairs should have at least 50 entries, got %d", len(captchaImagePairs))
	}
}

func TestCaptchaTextPairsKnownEntry(t *testing.T) {
	val, ok := captchaTextPairs["0044a9ac08865fb4c9b4984319d3ed31"]
	if !ok {
		t.Fatal("expected to find known hash in captchaTextPairs")
	}
	if val != "the Doughnut" {
		t.Fatalf("expected 'the Doughnut', got %q", val)
	}
}

func TestCaptchaImagePairsKnownEntry(t *testing.T) {
	val, ok := captchaImagePairs["the Apple"]
	if !ok {
		t.Fatal("expected to find 'the Apple' in captchaImagePairs")
	}
	if val != "c4dad21ade85a470f04e99669d90a77b" {
		t.Fatalf("expected 'c4dad21ade85a470f04e99669d90a77b', got %q", val)
	}
}

func TestSolveCaptchaInvalidBase64(t *testing.T) {
	result := SolveCaptcha("not-valid-base64!!!", "not-valid-base64!!!")
	if result < 0 || result > 3 {
		t.Fatalf("expected result between 0-3, got %d", result)
	}
}

func TestSolveCaptchaEmptyInput(t *testing.T) {
	result := SolveCaptcha("", "")
	if result < 0 || result > 3 {
		t.Fatalf("expected result between 0-3, got %d", result)
	}
}

func TestGetImageHash(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 10, 10))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	hash, err := getImageHashFromBase64(b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected 32-char hex hash, got %q", hash)
	}
}

func TestGetCaptchaImagesHashes(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 240, 60))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("failed to encode test image: %v", err)
	}
	b64 := base64.StdEncoding.EncodeToString(buf.Bytes())

	hashes, err := getCaptchaImagesHashesFromBase64(b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hashes) != 4 {
		t.Fatalf("expected 4 hashes, got %d", len(hashes))
	}
	for i, h := range hashes {
		if len(h) != 32 {
			t.Fatalf("hash[%d] expected 32 chars, got %d", i, len(h))
		}
	}
}
