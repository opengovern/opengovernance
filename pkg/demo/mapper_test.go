package demo

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMapper(t *testing.T) {
	assert.Equal(t, "RpVM", EncodeField("abcd"))
	assert.Equal(t, "abcd", DecodeField("RpVM"))
	str := "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	assert.Equal(t, str, DecodeField(EncodeField(str)))
	str = "---==-<?><:@#!)@#(%&^"
	assert.Equal(t, str, EncodeField(str))
	assert.Equal(t, str, DecodeField(str))
	str = "01-=)(*#&23456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRST><<?UVWXYZ"
	assert.Equal(t, str, DecodeField(EncodeField(str)))
}
