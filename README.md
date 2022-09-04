# golang_depoly_exercise

웹서버를 만들고 이를 github-action ci로 unit-test를 진행할 것이다.
이후에 AWS ECS에 올리는 과정까지 알아보자.

이 저장소는 https://youtube.com/playlist?list=PLy_6D98if3ULEtXtNSY_2qN21VCKgoQAE 에 나온 내용을 공부하여 일부를 정리한 것이다.

## Golang github action
### 간단한 테스트용 웹서버 만들기
데모 프로젝트 저장소는 여기서 확인하면 된다.
https://github.com/jiny0x01/golang_depoly_exercise


프로젝트 디렉토리를 생성하고 초기화 해준다.
> mkdir golang_deploy_exercise

> cd golang_deploy_exercise; go mod init golang_deploy_exercise; go mod tidy

웹서버에 쓸 gin과 unit test에 사용할 testify를 추가한다.

gin
> go get -u github.com/gin-gonic/gin

testify 
> go get github.com/stretchr/testify



```go
// main.go
package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type sumRequest struct {
	A int `json:"a" binding:"required"`
	B int `json:"b" binding:"required"`
}

func sum(a, b int) int {
	return a + b
}

func sumHandler(c *gin.Context) {
	var req sumRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error": err.Error(),
		})
		return
	}

	result := sum(req.A, req.B)

	c.JSON(http.StatusOK, gin.H{
		"result": result,
	})
}

func main() {
	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})
	r.POST("/sum", sumHandler)

	r.Run()
}
```

```go
//main_test.go
package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestSum(t *testing.T) {
	testCases := []struct {
		name          string
		body          gin.H
		checkResponse func(t *testing.T, recorder *httptest.ResponseRecorder)
	}{
		{
			name: "OK",
			body: gin.H{
				"a": 10,
				"b": 20,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusOK, recorder.Code)
			},
		},
		{
			name: "UnprocessableEntity",
			body: gin.H{
				"a": 10,
				"c": 50,
			},
			checkResponse: func(t *testing.T, recorder *httptest.ResponseRecorder) {
				require.Equal(t, http.StatusUnprocessableEntity, recorder.Code)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			router := gin.Default()
			router.POST("/sum", sumHandler)
			recorder := httptest.NewRecorder()

			data, err := json.Marshal(tc.body)
			require.NoError(t, err)

			url := "/sum"
			request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
			require.NoError(t, err)
			router.ServeHTTP(recorder, request)
			tc.checkResponse(t, recorder)
		})
	}
}
```

***/sum***에 POST 요청으로 ***a***와 ***b***를 보내면 a+b를 반환해주는 간단한 웹서버다.
아래 명령어로 전체 테스트를 진행할 수 있다.

> go test -v ./...
![[unit_test.png]]


이제 github repo에서 actions 탭에서 Go 템플릿을 살짝 수정할거다.
![[action1.png]]

```yaml
name: Run unit test

on:
  push:
    branches: [ "main" ]
  pull_request:
    branches: [ "main" ]

jobs:

  test:
    name: Test
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v3

    - name: Set up Go
      uses: actions/setup-go@v3
      with:
        go-version: 1.18

    - name: Test
      run: go test -v ./... # ./...는 현재 디렉토리 아래에 있는 모든 것을 의미
```


```

모든 파일들을 다 추가하고 push하면 된다.
```bash
git add .
git commit m -m "Add unit test"
git push
```

![[unit_test_result.png]]


#### workflow
workflow에는 job이 존재하며 한 job에는 여러 step이 있다.
각각의 job은 병렬로 동작하거나 순서에 맞춰 동작할 수 있다. 
하나의 job은 여러개의 step을 갖고 있을 수 있다.

##### step
각각의 step에는 1개 이상의 action이 존재한다. 
github action에서 알고 넘어가야할 부분은 job과 step이다.

 + https://docs.github.com/en/actions/using-jobs/using-jobs-in-a-workflow

앞서 본 unit test는 단순히 ubuntu 환경에서 golang을 set-up하고 go test를 실행한 것이 전부다.
추가적인 작업이 필요하다면 name을 정해주고 어떤 명령을 수행할지 run에서 정의해주면 된다.
```yaml
- name: build
  run: go build -v
```

action에 대해 더 깊게 이해하고 싶다면 아래 공식문서의 설명을 읽어보자
https://docs.github.com/en/actions/learn-github-actions/understanding-github-actions#the-components-of-github-actions


# ECR Build
위에서 만든 서버 애플리케이션을 ECR(Elastic Container Registry)에 배포 해볼 것이다. 

우선 Dockerfile을 작성해준다.
```Dockerfile
#Dockerfile
#Build stage 
FROM golang:1.18.5-alpine3.16 AS builder 
WORKDIR /app COPY . . 
RUN go build -o main main.go 

# Run stage 
FROM alpine:3.16 
WORKDIR /app COPY --from=builder /app/main . 
EXPOSE 8080 
CMD [ "/app/main" ]
```

FROM을 2개 쓰는 이유가 궁금하면 여기로 
+ (multi stage build)[https://cafemocamoca.tistory.com/320]

docker-compose를 사용하여 서버 애플리케이션을 실행할 것이다.

```yaml
# demo docker-compose
version: "3.9"
services:
  api:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    command: [ "/app/main" ]
```



# RDS Connect

#
