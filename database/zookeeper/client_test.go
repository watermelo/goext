package gxzookeeper

import (
	"strings"
	"sync"
	"testing"
	"time"
)

import (
	jerrors "github.com/juju/errors"
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/suite"
)

type ClientTestSuite struct {
	suite.Suite
	client *Client
}

func (suite *ClientTestSuite) SetupSuite() {
}

func (suite *ClientTestSuite) SetupTest() {
	conn, _, _ := zk.Connect([]string{"127.0.0.1:2181"}, 3e9)
	suite.client = NewClient(conn)
}

func (suite *ClientTestSuite) TearDownTest() {
	suite.client.ZkConn().Close()
}

func (suite *ClientTestSuite) TearDownSuite() {
}

func (suite *ClientTestSuite) TestClient_RegisterTempSeq() {
	suite.client.CreateZkPath("/test/hello/xx-")
	path, err := suite.client.RegisterTempSeq("/test/hello/xx-", []byte("v0"))
	suite.T().Logf("path:%s, error:%v", path, err)
	suite.Equal(nil, err)
	path, err = suite.client.RegisterTempSeq("/test/hello/xx-", []byte("v1"))
	suite.T().Logf("path:%s, error:%v", path, err)
	suite.Equal(nil, err)
	// time.Sleep(90e9)
}

func (suite *ClientTestSuite) TestClient_RegisterTempSeq() {
	path := "/test"
	err := suite.client.CreateZkPath(path)
	suite.Equal(nil, err, "CreateZkPath")

	path += "/hello"
	data := "world"
	tempPath, err := suite.client.RegisterTempSeq(path, []byte(data))
	suite.Equal(nil, err, "RegisterTempSeq")
	suite.Equal(true, strings.HasPrefix(tempPath, path), "tempPath:%s", tempPath)

	suite.client.DeleteZkPath(path)
}

func (suite *ClientTestSuite) TestClient_RegisterTemp() {
	path := "/getty-root"
	err := suite.client.CreateZkPath(path)
	suite.Equal(nil, err, "CreateZkPath")

	path += "/group%3Dbj-yizhuang%26protocol%3Dprotobuf%26role%3DSRT_Provider%26service" +
		"%3DTestService%26version%3Dv1.0"
	err = suite.client.CreateZkPath(path)
	suite.Equal(nil, err, "CreateZkPath")

	path += "/svr-node1"
	data := "world"
	tempPath, err := suite.client.RegisterTemp(path, []byte(data))
	suite.Equal(nil, err, "RegisterTemp")
	suite.Equal(true, strings.HasPrefix(tempPath, path), "tempPath:%s", tempPath)
}

func (suite *ClientTestSuite) TestClient_Leader() {
	var (
		err error
		wg  sync.WaitGroup
	)

	fn := func(i int, timeout time.Duration) {
		defer wg.Done()
		err = suite.client.Campaign("/test-lock/", timeout)
		if err == nil {
			time.Sleep(3e9)
			suite.client.Resign("/test-lock/")
		} else {
			suite.T().Logf("index:%d, err:%s", i, jerrors.ErrorStack(err))
		}
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go fn(i, time.Duration(i*3e9))
	}

	wg.Wait()
}

func (suite *ClientTestSuite) TestClient_LongLiveLeader() {
	var (
		err error
		wg  sync.WaitGroup
	)

	fn := func(i int, timeout time.Duration) {
		defer wg.Done()
		err = suite.client.Campaign("/test-leader/", timeout)
		if err == nil {
			time.Sleep(3e9)
			suite.client.Resign("/test-leader/")
		} else {
			suite.T().Logf("index:%d, err:%s", i, jerrors.ErrorStack(err))
		}
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)
		go fn(i, time.Duration(i*3e9))
	}
	wg.Add(1)
	go fn(4, time.Duration(0))

	wg.Wait()
}

func (suite *ClientTestSuite) TestClient_Lock() {
	path := "/test-client-lock/"
	err := suite.client.Lock(path)
	suite.Equal(nil, err)
	err = suite.client.Lock(path)
	suite.Equal(jerrors.Cause(err).Error(), ErrDeadlock.Error())

	err = suite.client.Unlock(path)
	suite.Equal(nil, err)
	err = suite.client.Unlock(path)
	suite.Equal(jerrors.Cause(err).Error(), ErrNotLocked.Error())

	err = suite.client.Lock(path)
	suite.Equal(nil, err)
	err = suite.client.Unlock(path)
	suite.Equal(nil, err)
}

func TestClientTestSuite(t *testing.T) {
	suite.Run(t, new(ClientTestSuite))
}

func TestGetSequenceNumber(t *testing.T) {
	prefix := "/test/seq-"
	num, err := getSequenceNumber(prefix+"1", prefix)
	if err != nil {
		t.Errorf("err should be nil")
	}
	if num != 1 {
		t.Errorf("num should be equal to 1")
	}

	num = 0
	num, err = getSequenceNumber(prefix, prefix)
	if err == nil {
		t.Errorf("err should nil be nil")
	}
	if num == 1 {
		t.Errorf("num should not be equal to 1")
	}
}

func TestGetMinSequenceNumber(t *testing.T) {
	prefix := "/test/seq-"
	seq, index, err := GetMinSequenceNumber(nil, prefix)
	if err == nil {
		t.Errorf("err should not be nil")
	}

	seq, index, err = GetMinSequenceNumber([]string{prefix + "1"}, prefix)
	if err != nil {
		t.Errorf("err should be nil")
	}
	if seq != 1 {
		t.Errorf("seq should be 1")
	}
	if index != 0 {
		t.Errorf("index should be 0")
	}

	seq, index, err = GetMinSequenceNumber([]string{prefix + "2", prefix + "1", prefix + "3"}, prefix)
	if err != nil {
		t.Errorf("err should be nil")
	}
	if seq != 1 {
		t.Errorf("seq should be 1")
	}
	if index != 1 {
		t.Errorf("index should be 1")
	}

	seq, index, err = GetMinSequenceNumber([]string{prefix + "2", prefix + "1", prefix + "3", prefix + "-1"}, prefix)
	if err != nil {
		t.Errorf("err should be nil")
	}
	if seq != -1 {
		t.Errorf("seq should be -1")
	}
	if index != 3 {
		t.Errorf("index should be 3")
	}
}

func TestGetMaxSequenceNumber(t *testing.T) {
	prefix := "/test/seq-"
	seq, index, err := GetMinSequenceNumber(nil, prefix)
	if err == nil {
		t.Errorf("err should not be nil")
	}

	seq, index, err = GetMaxSequenceNumber([]string{prefix + "1"}, prefix)
	if err != nil {
		t.Errorf("err should be nil")
	}
	if seq != 1 {
		t.Errorf("seq should be 1")
	}
	if index != 0 {
		t.Errorf("index should be 0")
	}

	seq, index, err = GetMaxSequenceNumber([]string{prefix + "4", prefix + "1", prefix + "3"}, prefix)
	if err != nil {
		t.Errorf("err should be nil")
	}
	if seq != 4 {
		t.Errorf("seq should be 2")
	}
	if index != 0 {
		t.Errorf("index should be 0")
	}

	seq, index, err = GetMaxSequenceNumber([]string{prefix + "2", prefix + "1", prefix + "3", prefix + "4"}, prefix)
	if err != nil {
		t.Errorf("err should be nil")
	}
	if seq != 4 {
		t.Errorf("seq should be 4")
	}
	if index != 3 {
		t.Errorf("index should be 3")
	}
}
