/**
 * Copyright 2015 @ S1N1 Team.
 * name : user_c
 * author : jarryliu
 * date : -- :
 * description :
 * history :
 */
package ols

import (
	"errors"
	"fmt"
	"github.com/atnet/gof"
	"github.com/atnet/gof/web"
	"github.com/atnet/gof/web/mvc"
	"go2o/src/app/util"
	"go2o/src/core/domain/interface/member"
	"go2o/src/core/infrastructure/domain"
	"go2o/src/core/service/dps"
	"go2o/src/core/variable"
	"strings"
)

var _ mvc.Filter = new(UserC)

type UserC struct {
	*BaseC
}

func (this *UserC) Login(ctx *web.Context) {
	p := this.BaseC.GetPartner(ctx)
	r := ctx.Request
	var tipStyle string
	var returnUrl string = r.URL.Query().Get("return_url")
	if len(returnUrl) == 0 {
		tipStyle = " hidden"
	}

	siteConf := this.GetSiteConf(ctx)
	this.BaseC.ExecuteTemplate(ctx, gof.TemplateDataMap{
		"partner":  p,
		"title":    "会员登录－" + siteConf.SubTitle,
		"conf":     siteConf,
		"tipStyle": tipStyle,
	},
		"views/shop/ols/{device}/login.html",
		"views/shop/ols/{device}/inc/header.html",
		"views/shop/ols/{device}/inc/footer.html")

}

func (this *UserC) Login_post(ctx *web.Context) {
	r := ctx.Request
	r.ParseForm()
	var result gof.Message
	partnerId := this.BaseC.GetPartnerId(ctx)
	usr, pwd := r.Form.Get("usr"), r.Form.Get("pwd")
	b, m, err := dps.MemberService.Login(partnerId, usr, pwd)
	if b {
		ctx.Session().Set("member", m)
		ctx.Session().Save()
	} else {
		result.Result = false
		if err != nil {
			result.Message = err.Error()
		} else {
			result.Message = "登陆失败"
		}
	}
	this.BaseC.ResultOutput(ctx, result)
}

func (this *UserC) Register(ctx *web.Context) {
	p := this.BaseC.GetPartner(ctx)

	siteConf := this.BaseC.GetSiteConf(ctx)
	this.BaseC.ExecuteTemplate(ctx, gof.TemplateDataMap{
		"partner": p,
		"title":   "会员注册－" + siteConf.SubTitle,
		"conf":    siteConf,
	},
		"views/shop/ols/{device}/register.html",
		"views/shop/ols/{device}/inc/header.html",
		"views/shop/ols/{device}/inc/footer.html")

}

func (this *UserC) ValidUsr_post(ctx *web.Context) {
	r, w := ctx.Request, ctx.ResponseWriter
	r.ParseForm()
	usr := r.FormValue("usr")
	b := dps.MemberService.CheckUsrExist(usr)
	if !b {
		w.Write([]byte(`{"result":true}`))
	} else {
		w.Write([]byte(`{"result":false}`))
	}
}

func (this *UserC) Valid_invitation_post(ctx *web.Context) {
	ctx.Request.ParseForm()
	memberId := dps.MemberService.GetMemberIdByInvitationCode(ctx.Request.FormValue("code"))

	var message string
	isOk := memberId != 0

	if !isOk {
		message = "推荐人无效"
	}
	this.BaseC.ResultOutput(ctx, gof.Message{Result: isOk, Message: message})
}

func (this *UserC) PostRegisterInfo_post(ctx *web.Context) {
	ctx.Request.ParseForm()
	var member member.ValueMember
	web.ParseFormToEntity(ctx.Request.Form, &member)
	if i := strings.Index(ctx.Request.RemoteAddr, ":"); i != -1 {
		member.RegIp = ctx.Request.RemoteAddr[:i]
	}

	var memberId int
	var err error

	if len(member.Usr) == 0 || len(member.Pwd) == 0 {
		err = errors.New("注册信息不完整")
	} else {
		member.Pwd = domain.Md5MemberPwd(member.Usr, member.Pwd)
		memberId, err = dps.MemberService.SaveMember(&member)
		if err == nil {
			invId := dps.MemberService.GetMemberIdByInvitationCode(ctx.Request.FormValue("inviCode"))
			err = dps.MemberService.SaveRelation(memberId, "", invId,
				this.BaseC.GetPartnerId(ctx))
		}
	}

	if err != nil {
		this.BaseC.ResultOutput(ctx, gof.Message{Message: "注册失败！错误：" + err.Error()})
	} else {
		this.BaseC.ResultOutput(ctx, gof.Message{Result: true})
	}
}

// 跳转到会员中心
// url : /user/jump_m
func (this *UserC) JumpToMCenter(ctx *web.Context) {
	w := ctx.ResponseWriter
	m := this.BaseC.GetMember(ctx)
	var location string
	if m == nil {
		location = "/user/login?return_url=/user/jump_m"
	} else {
		location = fmt.Sprintf("http://%s.%s/login/partner_connect?device=%s&sessionId=%s&mid=%d&token=%s",
			variable.DOMAIN_MEMBER_PREFIX,
			ctx.App.Config().GetString(variable.ServerDomain),
			util.GetBrownerDevice(ctx),
			ctx.Session().GetSessionId(),
			m.Id,
			m.DynamicToken,
		)
	}
	w.Header().Add("Location", location)
	w.WriteHeader(302)
}

// 退出
func (this *UserC) Logout(ctx *web.Context) {
	ctx.Session().Set("member", nil)
	ctx.Session().Save()
	ctx.ResponseWriter.Write([]byte(fmt.Sprintf(`<html><head><title>正在退出...</title></head><body>
			3秒后将自动返回到首页... <br />
			<iframe src="http://%s.%s/login/partner_disconnect" width="0" height="0" frameBorder="0"></iframe>
			<script>window.onload=function(){location.replace('/')}</script></body></html>`,
		variable.DOMAIN_MEMBER_PREFIX,
		ctx.App.Config().GetString(variable.ServerDomain),
	)))
}
