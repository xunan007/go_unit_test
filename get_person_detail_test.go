// go test -gcflags=all=-l -coverprofile=coverage.out
// go tool cover -html=coverage.out
package unit

import (
	"errors"
	"github.com/agiledragon/gomonkey"
	"github.com/golang/mock/gomock"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetPersonDetail(t *testing.T) {
	type args struct {
		username string
	}
	tests := []struct {
		name    string
		args    args
		want    *PersonDetail
		wantErr bool
	}{
		{name: "invalid username", args: args{username: "steven xxx"}, want: nil, wantErr: true},
		{name: "invalid email", args: args{username: "invalid_email"}, want: nil, wantErr: true},
		{name: "throw err", args: args{username: "throw_err"}, want: nil, wantErr: true},
		{name: "valid return", args: args{username: "steven"}, want: &PersonDetail{Username: "steven", Email: "12345678@qq.com"}, wantErr: false},
	}

	// 为函数打桩序列
	// 使用 gomonkey 打函数桩序列
	// 第一个用例不会调用 getPersonDetailRedis，所以只需要 3 个值
	outputs := []gomonkey.OutputCell{
		{
			Values: gomonkey.Params{&PersonDetail{Username: "invalid_email", Email: "test.com"}, nil},
		},
		{
			Values: gomonkey.Params{nil, errors.New("request err")},
		},
		{
			Values: gomonkey.Params{&PersonDetail{Username: "steven", Email: "12345678@qq.com"}, nil},
		},
	}
	patches := gomonkey.ApplyFuncSeq(getPersonDetailRedis, outputs)
	// 执行完毕后释放桩序列
	defer patches.Reset()

	for _, tt := range tests {
		got, err := GetPersonDetail(tt.args.username)
		assert.Equal(t, tt.want, got)
		assert.Equal(t, tt.wantErr, err != nil)
	}
}

func Test_checkEmail(t *testing.T) {
	type args struct {
		email string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "email valid",
			args: args{
				email: "1234567@qq.com",
			},
			want: true,
		},
		{
			name: "email invalid",
			args: args{
				email: "test.com",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		got := checkEmail(tt.args.email)
		assert.Equal(t, tt.want, got)
	}
}

func Test_checkUsername(t *testing.T) {
	type args struct {
		username string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "valid username",
			args: args{username: "steven"},
			want: true,
		},
		{
			name: "invalid username",
			args: args{username: "steven xxx"},
			want: false,
		},
	}
	for _, tt := range tests {
		got := checkUsername(tt.args.username)
		assert.Equal(t, tt.want, got)
	}
}

func Test_getPersonDetailRedis(t *testing.T) {
	tests := []struct {
		name    string
		want    *PersonDetail
		wantErr bool
	}{
		{name: "redis.Do err", want: nil, wantErr: true},
		{name: "json.Unmarshal err", want: nil, wantErr: true},
		{name: "success", want: &PersonDetail{
			Username: "steven",
			Email:    "1234567@qq.com",
		}, wantErr: false},
	}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// 1. 生成符合 redis.Conn 接口的 mockConn
	mockConn := NewMockConn(ctrl)

	// 2. 给接口打桩序列
	gomock.InOrder(
		mockConn.EXPECT().Do("GET", gomock.Any()).Return("", errors.New("redis.Do err")),
		mockConn.EXPECT().Close().Return(nil),
		mockConn.EXPECT().Do("GET", gomock.Any()).Return("123", nil),
		mockConn.EXPECT().Close().Return(nil),
		mockConn.EXPECT().Do("GET", gomock.Any()).Return([]byte(`{"username": "steven", "email": "1234567@qq.com"}`), nil),
		mockConn.EXPECT().Close().Return(nil),
	)

	// 3. 给 redis.Dail 函数打桩
	outputs := []gomonkey.OutputCell{
		{
			Values: gomonkey.Params{mockConn, nil},
			Times:  3, // 3 个用例
		},
	}
	patches := gomonkey.ApplyFuncSeq(redis.Dial, outputs)
	// 执行完毕之后释放桩序列
	defer patches.Reset()

	// 4. 断言
	for _, tt := range tests {
		actual, err := getPersonDetailRedis(tt.name)
		// 注意，equal 函数能够对结构体进行 deap diff
		assert.Equal(t, tt.want, actual)
		assert.Equal(t, tt.wantErr, err != nil)
	}
}
