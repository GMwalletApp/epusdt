package comm

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/assimon/luuu/config"
	"github.com/assimon/luuu/model/response"
	"github.com/assimon/luuu/model/service"
	"github.com/labstack/echo/v4"
)

// CheckoutCounter 收银台
func (c *BaseCommController) CheckoutCounter(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	resp, err := service.GetCheckoutCounterByTradeId(tradeId)

	type checkoutCounterPageData struct {
		response.CheckoutCounterResponse
		PaymentOptionsJSON template.JS
	}
	buildPageData := func(resp response.CheckoutCounterResponse) (checkoutCounterPageData, error) {
		paymentOptionsJSON, err := json.Marshal(resp.PaymentOptions)
		if err != nil {
			return checkoutCounterPageData{}, err
		}
		return checkoutCounterPageData{
			CheckoutCounterResponse: resp,
			PaymentOptionsJSON:      template.JS(string(paymentOptionsJSON)),
		}, nil
	}

	if err != nil {
		if err == service.ErrOrder {
			tmpl, err := template.ParseFiles(filepath.Join(config.StaticFilePath, "index.html"))
			if err != nil {
				return ctx.String(http.StatusOK, err.Error())
			}
			pageData, err := buildPageData(response.CheckoutCounterResponse{})
			if err != nil {
				return ctx.String(http.StatusOK, err.Error())
			}
			return tmpl.Execute(ctx.Response(), pageData)
		}
		return ctx.String(http.StatusOK, err.Error())
	}
	tmpl, err := template.ParseFiles(filepath.Join(config.StaticFilePath, "index.html"))
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}

	jsonByte, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}
	fmt.Printf("%v\n", string(jsonByte))

	pageData, err := buildPageData(*resp)
	if err != nil {
		return ctx.String(http.StatusOK, err.Error())
	}
	return tmpl.Execute(ctx.Response(), pageData)
}

// CheckStatus 支付状态检测
func (c *BaseCommController) CheckStatus(ctx echo.Context) (err error) {
	tradeId := ctx.Param("trade_id")
	order, err := service.GetOrderInfoByTradeId(tradeId)
	if err != nil {
		return c.FailJson(ctx, err)
	}
	resp := response.CheckStatusResponse{
		TradeId: order.TradeId,
		Status:  order.Status,
	}
	return c.SucJson(ctx, resp)
}
