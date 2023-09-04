package demo

import "github.com/labstack/echo/v4"

func EncodeResponseData(ctx echo.Context, value string) string {
	if IsDemo(ctx) {
		return EncodeField(value)
	}
	return value
}

func EncodeResponseArray(ctx echo.Context, value []string) []string {
	for i, v := range value {
		value[i] = EncodeResponseData(ctx, v)
	}
	return value
}

func DecodeRequestData(ctx echo.Context, value string) string {
	if IsDemo(ctx) {
		return DecodeField(value)
	}
	return value
}

func DecodeRequestArray(ctx echo.Context, value []string) []string {
	for i, v := range value {
		value[i] = DecodeRequestData(ctx, v)
	}
	return value
}

func IsDemo(ctx echo.Context) bool {
	demoHeader := ctx.Request().Header.Get("X-Kaytu-Demo")
	return len(demoHeader) > 0 && demoHeader == "true"
}
