package web

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	regexp "github.com/dlclark/regexp2"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"geektime/webook/internal/domain"
	"geektime/webook/internal/service"
)

var (
	ErrUserEmailDuplicate = service.ErrUserEmailDuplicate
)

type UserHandler struct {
	svc            *service.UserService
	emailRegexp    *regexp.Regexp
	passwordRegexp *regexp.Regexp
}

func NewUserHandler(svc *service.UserService) *UserHandler {
	const (
		emailRegexPattern = "^\\w+([-+.]\\w+)*@\\w+([-.]\\w+)*\\.\\w+([-.]\\w+)*$"
		// 和上面比起来，用 ` 看起来就比较清爽
		passwordRegexPattern = `^(?=.*[A-Za-z])(?=.*\d)(?=.*[$@$!%*#?&])[A-Za-z\d$@$!%*#?&]{8,}$`
	)
	return &UserHandler{
		svc:            svc,
		emailRegexp:    regexp.MustCompile(emailRegexPattern, regexp.None),
		passwordRegexp: regexp.MustCompile(passwordRegexPattern, regexp.None),
	}
}

func (u *UserHandler) RegisterRoutes(server *gin.Engine) {
	ug := server.Group("/users")
	{
		ug.POST("/signup", u.SignUp)
		// ug.POST("/login", u.Login)
		ug.POST("/login", u.LoginJWT)
		ug.POST("/edit", u.Edit)
		// ug.GET("/profile", u.Profile)
		ug.GET("/profile", u.ProfileJWT)
	}
}

func (u *UserHandler) SignUp(ctx *gin.Context) {
	type SignUpReq struct {
		Email           string `json:"email"`
		Password        string `json:"password"`
		ConfirmPassword string `json:"confirmPassword"`
	}
	var req SignUpReq
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 验证邮箱格式
	if ok, err := u.emailRegexp.MatchString(req.Email); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	} else if !ok {
		ctx.String(http.StatusOK, "邮箱格式错误")
		return
	}

	// 验证密码格式
	if ok, err := u.passwordRegexp.MatchString(req.Password); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	} else if !ok {
		ctx.String(http.StatusOK, "密码至少8个字符，至少1个字母，1个数字和1个特殊字符")
		return
	}

	// 验证两次密码是否一致
	if req.Password != req.ConfirmPassword {
		ctx.String(http.StatusOK, "两次输入密码不一致")
		return
	}

	// 数据库操作
	err := u.svc.SignUp(ctx, domain.User{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		if errors.Is(err, ErrUserEmailDuplicate) {
			ctx.String(http.StatusOK, "邮箱重复")
			return
		} else {
			ctx.String(http.StatusOK, "系统错误")
			return
		}
	}

	// 注册成功
	ctx.String(http.StatusOK, "注册成功")
}

func (u *UserHandler) Login(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	user, err := u.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUserOrPassword) {
			ctx.String(http.StatusOK, "账户或密码错误")
			return
		}
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	// 设置session
	// 步骤2
	sess := sessions.Default(ctx)
	sess.Set("userId", user.Id)
	sess.Options(sessions.Options{
		// Secure: true,
		// HttpOnly: true,
		MaxAge: 60,
	})
	sess.Save()

	ctx.String(http.StatusOK, "登录成功")
}

type UserClaims struct {
	jwt.RegisteredClaims
	// 声明要放进 token 中的数据
	Uid       int64
	UserAgent string
}

func (u *UserHandler) LoginJWT(ctx *gin.Context) {
	type LoginReq struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	var req LoginReq
	if err := ctx.Bind(&req); err != nil {
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	user, err := u.svc.Login(ctx, req.Email, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidUserOrPassword) {
			ctx.String(http.StatusOK, "账户或密码错误")
			return
		}
		ctx.String(http.StatusOK, "系统错误")
		return
	}

	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Minute * 30)),
		},
		Uid:       user.Id,
		UserAgent: ctx.Request.UserAgent(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, claims)
	// token := jwt.New(jwt.SigningMethodHS512)
	tokenStr, err := token.SignedString([]byte("aY3?fW6+kK9~mX7!yQ5|wS7%vR8_lO1"))
	if err != nil {
		ctx.String(http.StatusInternalServerError, "系统错误")
		return
	}
	ctx.Header("x-jwt-token", tokenStr)
	fmt.Println(tokenStr)
	fmt.Println(user)

	ctx.String(http.StatusOK, "登录成功")
}

func (u *UserHandler) Edit(ctx *gin.Context) {

}

func (u *UserHandler) Profile(ctx *gin.Context) {
	ctx.String(http.StatusOK, "这是你的响应")
	return
}

func (u *UserHandler) ProfileJWT(ctx *gin.Context) {
	// 不 ok 的话 c == nil, 下面类型断言会报错
	// c, ok := ctx.Get("claims")
	// if !ok {
	// 	ctx.String(http.StatusOK, "系统错误")
	// 	return
	// }

	c, _ := ctx.Get("claims")
	claims, ok := c.(*UserClaims)
	if !ok {
		ctx.String(http.StatusOK, "系统错误")
		return
	}
	fmt.Println(claims.Uid)
	ctx.String(http.StatusOK, "your profile")
}
