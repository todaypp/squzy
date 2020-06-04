package config

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	t.Run("Shoud: return default value", func(t *testing.T) {
		s := New()
		assert.Equal(t, s.GetPort(), defaultPort)
		assert.Equal(t, s.GetStorageHost(), "")
		assert.Equal(t, s.GetMongoURI(), "")
		assert.Equal(t, s.GetMongoDb(), defaultMongoDb)
		assert.Equal(t, s.GetStorageTimeout(), defaultStorageTimeout)
		assert.Equal(t, s.GetMongoCollection(), defaultCollection)
		assert.Equal(t, s.GetTracingHeader(), defaultTracingHeader)
	})
}



func TestCfg_GetPort(t *testing.T) {
	t.Run("Should: return from env", func(t *testing.T) {
		os.Setenv(ENV_PORT, "11124")
		s := New()
		assert.Equal(t, s.GetPort(), int32(11124))
	})
}

func TestCfg_GetStorageHost(t *testing.T) {
	t.Run("Should: return from env", func(t *testing.T) {
		os.Setenv(ENV_STORAGE_HOST, "11124")
		s := New()
		assert.Equal(t, s.GetStorageHost(), "11124")
	})
}

func TestCfg_GetStorageTimeout(t *testing.T) {
	t.Run("Should: return from env", func(t *testing.T) {
		os.Setenv(ENV_STORAGE_TIMEOUT, "11")
		s := New()
		assert.Equal(t, s.GetStorageTimeout(), time.Second*11)
	})
}

func TestCfg_GetMongoCollection(t *testing.T) {
	t.Run("Should: return from env", func(t *testing.T) {
		os.Setenv(ENV_MONGO_COLLECTION, "11124")
		s := New()
		assert.Equal(t, s.GetMongoCollection(), "11124")
	})
}

func TestCfg_GetMongoDb(t *testing.T) {
	t.Run("Should: return from env", func(t *testing.T) {
		os.Setenv(ENV_MONGO_DB, "11124")
		s := New()
		assert.Equal(t, s.GetMongoDb(), "11124")
	})
}

func TestCfg_GetMongoURI(t *testing.T) {
	t.Run("Should: return from env", func(t *testing.T) {
		os.Setenv(ENV_MONGO_URI, "11124")
		s := New()
		assert.Equal(t, s.GetMongoURI(), "11124")
	})
}

func TestCfg_GetTracingHeader(t *testing.T) {
	t.Run("Should: return from env", func(t *testing.T) {
		os.Setenv(ENV_TRACING_HEADER, "11124")
		s := New()
		assert.Equal(t, s.GetTracingHeader(), "11124")
	})
}
