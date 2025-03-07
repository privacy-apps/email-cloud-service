package mail

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	_ "fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/flashmob/go-guerrilla"
	"github.com/flashmob/go-guerrilla/backends"
	"github.com/flashmob/go-guerrilla/log"
	test "github.com/flashmob/go-guerrilla/tests"
	"github.com/flashmob/go-guerrilla/tests/testcert"
	maildir_processor "github.com/flashmob/maildir-processor"
	"github.com/spf13/cobra"
)

var configJsonA = `
{
    "log_file" : "./_test/testlog",
    "log_level" : "debug",
    "pid_file" : "./pidfile.pid",
    "allowed_hosts": [
      "guerrillamail.com",
      "guerrillamailblock.com",
      "sharklasers.com",
      "guerrillamail.net",
      "guerrillamail.org"
    ],

    "backend_config": {
        "log_received_mails": true
    },
    "servers" : [
        {
            "is_enabled" : true,
            "host_name":"mail.test.com",
            "max_size": 1000000,
			"tls" : {
 				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
            	"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
				"start_tls_on":true,
            	"tls_always_on":false
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:3535",
            "max_clients": 1000,
            "log_file" : "_test/testlog"
        },
        {
            "is_enabled" : false,
            "host_name":"enable.test.com",
            "max_size": 1000000, 
			"tls" : {
				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
            	"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
				"start_tls_on":true,
            	"tls_always_on":false
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:2228",
            "max_clients": 1000,
            "log_file" : "_test/testlog"
        }
    ]
}
`

// backend config changed, log_received_mails is false
var configJsonB = `
{
    "log_file" : "./_test/testlog",
    "log_level" : "debug",
    "pid_file" : "./pidfile2.pid",
    "allowed_hosts": [
      "guerrillamail.com",
      "guerrillamailblock.com",
      "sharklasers.com",
      "guerrillamail.net",
      "guerrillamail.org"
    ],

    "backend_config": {
        "log_received_mails": false
    },
    "servers" : [
        {
            "is_enabled" : true,
            "host_name":"mail.test.com",
            "max_size": 1000000,
			"tls" : {
				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
				"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
				"start_tls_on":true,
            	"tls_always_on":false
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:3535",
            "max_clients": 1000,
            "log_file" : "_test/testlog"
        }
    ]
}
`

// backend_name changed, is guerrilla-redis-db + added a server
var configJsonC = `
{
    "log_file" : "./_test/testlog",
    "log_level" : "debug",
    "pid_file" : "./pidfile.pid",
    "allowed_hosts": [
      "guerrillamail.com",
      "guerrillamailblock.com",
      "sharklasers.com",
      "guerrillamail.net",
      "guerrillamail.org"
    ],

    "backend_config" :
        {
            "mysql_db":"gmail_mail",
            "mysql_host":"127.0.0.1:3306",
            "mysql_pass":"ok",
            "mysql_user":"root",
            "mail_table":"new_mail",
            "redis_interface" : "127.0.0.1:6379",
            "redis_expire_seconds" : 7200,
            "save_workers_size" : 3,
            "primary_mail_host":"sharklasers.com"
        },
    "servers" : [
        {
            "is_enabled" : true,
            "host_name":"mail.test.com",
            "max_size": 1000000,
			"tls" : {
				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
				"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
				"start_tls_on":true,
				"tls_always_on":false
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:3535",
            "max_clients": 1000,
            "log_file" : "./_test/testlog"
        },
        {
            "is_enabled" : true,
            "host_name":"mail.test.com",
            "max_size":1000000,
			"tls" : {
				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
            	"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
				"start_tls_on":false,
            	"tls_always_on":true
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:465",
            "max_clients":500,
            "log_file" : "./_test/testlog"
        }
    ]
}
`

// adds 127.0.0.1:4655, a secure server
var configJsonD = `
{
    "log_file" : "./_test/testlog",
    "log_level" : "debug",
    "pid_file" : "./pidfile.pid",
    "allowed_hosts": [
      "guerrillamail.com",
      "guerrillamailblock.com",
      "sharklasers.com",
      "guerrillamail.net",
      "guerrillamail.org"
    ],
    "backend_config": {
        "log_received_mails": false
    },
    "servers" : [
        {
            "is_enabled" : true,
            "host_name":"mail.test.com",
            "max_size": 1000000, 
			"tls" : {
				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
            	"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
 				"start_tls_on":true,
            	"tls_always_on":false
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:2552",
            "max_clients": 1000,
            "log_file" : "./_test/testlog"
        },
        {
            "is_enabled" : true,
            "host_name":"secure.test.com",
            "max_size":1000000,
			"tls" : {
				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
            	"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
				"start_tls_on":false,
            	"tls_always_on":true
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:4655",
            "max_clients":500,
            "log_file" : "./_test/testlog"
        }
    ]
}
`

var configJsonE = `
{
    "log_file" : "./_test/testlog",
    "log_level" : "debug",
    "pid_file" : "./pidfile.pid",
    "allowed_hosts": [
      "guerrillamail.com",
      "guerrillamailblock.com",
      "sharklasers.com",
      "guerrillamail.net",
      "guerrillamail.org",
      "grr.la"
    ],

    "backend_config": {
        "save_process": "HeadersParser|Debugger|Hasher|Header|MailDir",
        "validate_process": "MailDir",
		"maildir_user_map" : "test=-1:-1",
		"maildir_path" : "_test/Maildir",
		"save_workers_size" : 1,
		"primary_mail_host":"sharklasers.com",
		"log_received_mails" : false
    },
    "servers" : [
        {
            "is_enabled" : true,
            "host_name":"mail.test.com",
            "max_size": 1000000,
			"tls" : {
				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
            	"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
				"start_tls_on":true,
            	"tls_always_on":false
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:3535",
            "max_clients": 1000,
            "log_file" : "./_test/testlog"
        },
        {
            "is_enabled" : false,
            "host_name":"enable.test.com",
            "max_size": 1000000,
            
			"tls" : {
				"private_key_file":"_test/mail2.guerrillamail.com.key.pem",
            	"public_key_file":"_test/mail2.guerrillamail.com.cert.pem",
				"start_tls_on":true,
            	"tls_always_on":false
			},
            "timeout":180,
            "listen_interface":"127.0.0.1:2228",
            "max_clients": 1000,
            "log_file" : "./_test/testlog"
        }
    ]
}
`

const testPauseDuration = time.Millisecond * 600

// reload config
func sigHup() {
	if data, err := ioutil.ReadFile("pidfile.pid"); err == nil {
		mainlog.Infof("pid read is %s", data)
		ecmd := exec.Command("kill", "-HUP", string(data))
		_, err = ecmd.Output()
		if err != nil {
			mainlog.Infof("could not SIGHUP", err)
		}
	} else {
		mainlog.WithError(err).Info("sighup - Could not read pidfle")
	}

}

// shutdown after calling serve()
func sigKill() {
	if data, err := ioutil.ReadFile("pidfile.pid"); err == nil {
		mainlog.Infof("pid read is %s", data)
		ecmd := exec.Command("kill", string(data))
		_, err = ecmd.Output()
		if err != nil {
			mainlog.Infof("could not sigkill", err)
		}
	} else {
		mainlog.WithError(err).Info("sigKill - Could not read pidfle")
	}

}

// make sure that we get all the config change events
func TestCmdConfigChangeEvents(t *testing.T) {

	oldconf := &guerrilla.AppConfig{}
	oldconf.Load([]byte(configJsonA))

	newconf := &guerrilla.AppConfig{}
	newconf.Load([]byte(configJsonB))

	newerconf := &guerrilla.AppConfig{}
	newerconf.Load([]byte(configJsonC))

	expectedEvents := map[guerrilla.Event]bool{
		guerrilla.EventConfigBackendConfig: false,
		guerrilla.EventConfigServerNew:     false,
	}
	mainlog, _ = log.GetLogger(log.OutputOff.String(), log.DebugLevel.String())

	bcfg := backends.BackendConfig{"log_received_mails": true}
	backend, err := backends.New(bcfg, mainlog)
	app, err := guerrilla.New(oldconf, backend, mainlog)
	if err != nil {
		mainlog.Info("Failed to create new app", err)
	}
	toUnsubscribe := map[guerrilla.Event]func(c *guerrilla.AppConfig){}
	toUnsubscribeS := map[guerrilla.Event]func(c *guerrilla.ServerConfig){}

	for event := range expectedEvents {
		// Put in anon func since range is overwriting event
		func(e guerrilla.Event) {

			if strings.Index(e.String(), "server_change") == 0 {
				f := func(c *guerrilla.ServerConfig) {
					expectedEvents[e] = true
				}
				app.Subscribe(event, f)
				toUnsubscribeS[event] = f
			} else {
				f := func(c *guerrilla.AppConfig) {
					expectedEvents[e] = true
				}
				app.Subscribe(event, f)
				toUnsubscribe[event] = f
			}

		}(event)
	}

	// emit events
	newconf.EmitChangeEvents(oldconf, app)
	newerconf.EmitChangeEvents(newconf, app)
	// unsubscribe
	for unevent, unfun := range toUnsubscribe {
		app.Unsubscribe(unevent, unfun)
	}

	for event, val := range expectedEvents {
		if val == false {
			t.Error("Did not fire config change event:", event)
			t.FailNow()
			break
		}
	}
	// cleanup
	os.Truncate("./_test/testlog", 0)

}

func init() {
	if err := os.MkdirAll("./_test/", 0755); err != nil {
		wd, _ := os.Getwd()
		fmt.Println("could not create test dir:", err, " wd:", wd)
	}
}

// start server, change config, send SIG HUP, confirm that the pidfile changed & backend reloaded
func TestStart(t *testing.T) {
	//if err := os.MkdirAll("./_test/", 0755); err != nil {
	//	wd, _ := os.Getwd()
	//	t.Fatal("could not create test dir:", err, " wd:", wd)
	//}
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "_test/")

	mainlog, _ = log.GetLogger("./_test/testlog", log.DebugLevel.String())

	ioutil.WriteFile("configJsonA.json", []byte(configJsonA), 0644)
	cmd := &cobra.Command{}
	configPath = "configJsonA.json"
	var serveWG sync.WaitGroup
	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration)
	data, err := ioutil.ReadFile("pidfile.pid")
	if err != nil {
		t.Error("error reading pidfile.pid", err)
		t.FailNow()
	}
	_, err = strconv.Atoi(string(data))
	if err != nil {
		t.Error("could not parse pidfile.pid", err)
		t.FailNow()
	}

	// change the config file
	ioutil.WriteFile("configJsonA.json", []byte(configJsonB), 0644)

	// test SIGHUP via the kill command
	// Would not work on windows as kill is not available.
	// TODO: Implement an alternative test for windows.
	if runtime.GOOS != "windows" {
		ecmd := exec.Command("kill", "-HUP", string(data))
		_, err = ecmd.Output()
		if err != nil {
			t.Error("could not SIGHUP", err)
			t.FailNow()
		}
		time.Sleep(testPauseDuration) // allow sighup to do its job
		// did the pidfile change as expected?
		if _, err := os.Stat("./pidfile2.pid"); os.IsNotExist(err) {
			t.Error("pidfile not changed after sighup SIGHUP", err)
		}
	}
	// send kill signal and wait for exit
	sigKill()
	serveWG.Wait()

	// did backend started as expected?
	fd, err := os.Open("./_test/testlog")
	if err != nil {
		t.Error(err)
	}
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "new backend started"); i < 0 {
			t.Error("Dummy backend not restared")
		}
	}

	// cleanup
	//os.Truncate("_test/testlog", 0)
	//os.Remove("configJsonA.json")
	os.Remove("./pidfile.pid")
	os.Remove("./pidfile2.pid")

}

// Start with configJsonA.json,
// then add a new server to it (127.0.0.1:2526),
// then SIGHUP (to reload config & trigger config update events),
// then connect to it & HELO.
func TestServerAddEvent(t *testing.T) {
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "_test/")
	mainlog, _ = log.GetLogger("./_test/testlog", log.DebugLevel.String())
	// start the server by emulating the serve command
	ioutil.WriteFile("configJsonA.json", []byte(configJsonA), 0644)
	cmd := &cobra.Command{}
	configPath = "configJsonA.json"
	var serveWG sync.WaitGroup
	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration) // allow the server to start
	// now change the config by adding a server
	conf := &CmdConfig{}                                 // blank one
	conf.Load([]byte(configJsonA))                       // load configJsonA
	newServer := conf.Servers[0]                         // copy the first server config
	newServer.ListenInterface = "127.0.0.1:2526"         // change it
	newConf := conf                                      // copy the cmdConfg
	newConf.Servers = append(newConf.Servers, newServer) // add the new server
	if jsonbytes, err := json.Marshal(newConf); err == nil {
		//fmt.Println(string(jsonbytes))
		ioutil.WriteFile("configJsonA.json", []byte(jsonbytes), 0644)
	}
	// send a sighup signal to the server
	sigHup()
	time.Sleep(testPauseDuration) // pause for config to reload

	if conn, buffin, err := test.Connect(newServer, 20); err != nil {
		t.Error("Could not connect to new server", newServer.ListenInterface)
	} else {
		if result, err := test.Command(conn, buffin, "HELO example.com"); err == nil {
			expect := "250 mail.test.com Hello"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			}
		} else {
			t.Error(err)
		}
	}

	// send kill signal and wait for exit
	sigKill()
	serveWG.Wait()

	// did backend started as expected?
	fd, _ := os.Open("./_test/testlog")
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "New server added [127.0.0.1:2526]"); i < 0 {
			t.Error("Did not add [127.0.0.1:2526], most likely because Bus.Subscribe(\"server_change:new_server\" didnt fire")
		}
	}
	// cleanup
	os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonA.json")
	os.Remove("./pidfile.pid")

}

// Start with configJsonA.json,
// then change the config to enable 127.0.0.1:2228,
// then write the new config,
// then SIGHUP (to reload config & trigger config update events),
// then connect to 127.0.0.1:2228 & HELO.
func TestServerStartEvent(t *testing.T) {
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "_test/")
	mainlog, _ = log.GetLogger("./_test/testlog", log.DebugLevel.String())
	// start the server by emulating the serve command
	ioutil.WriteFile("configJsonA.json", []byte(configJsonA), 0644)
	cmd := &cobra.Command{}
	configPath = "configJsonA.json"
	var serveWG sync.WaitGroup
	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration)
	// now change the config by adding a server
	conf := &CmdConfig{}           // blank one
	conf.Load([]byte(configJsonA)) // load configJsonA

	newConf := conf // copy the cmdConfg
	newConf.Servers[1].IsEnabled = true
	if jsonbytes, err := json.Marshal(newConf); err == nil {
		//fmt.Println(string(jsonbytes))
		ioutil.WriteFile("configJsonA.json", []byte(jsonbytes), 0644)
	} else {
		t.Error(err)
	}
	// send a sighup signal to the server
	sigHup()
	time.Sleep(testPauseDuration) // pause for config to reload

	if conn, buffin, err := test.Connect(newConf.Servers[1], 20); err != nil {
		t.Error("Could not connect to new server", newConf.Servers[1].ListenInterface)
	} else {
		if result, err := test.Command(conn, buffin, "HELO example.com"); err == nil {
			expect := "250 enable.test.com Hello"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			}
		} else {
			t.Error(err)
		}
	}
	// send kill signal and wait for exit
	sigKill()
	serveWG.Wait()
	// did backend started as expected?
	fd, _ := os.Open("./_test/testlog")
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "Starting server [127.0.0.1:2228]"); i < 0 {
			t.Error("did not add [127.0.0.1:2228], most likely because Bus.Subscribe(\"server_change:start_server\" didnt fire")
		}
	}
	// cleanup
	os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonA.json")
	os.Remove("./pidfile.pid")

}

// Start with configJsonA.json,
// then change the config to enable 127.0.0.1:2228,
// then write the new config,
// then SIGHUP (to reload config & trigger config update events),
// then connect to 127.0.0.1:2228 & HELO.
// then change the config to dsiable 127.0.0.1:2228,
// then SIGHUP (to reload config & trigger config update events),
// then connect to 127.0.0.1:2228 - it should not connect

func TestServerStopEvent(t *testing.T) {
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "_test/")
	mainlog, _ = log.GetLogger("./_test/testlog", log.DebugLevel.String())
	// start the server by emulating the serve command
	ioutil.WriteFile("configJsonA.json", []byte(configJsonA), 0644)
	cmd := &cobra.Command{}
	configPath = "configJsonA.json"
	var serveWG sync.WaitGroup
	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration)
	// now change the config by enabling a server
	conf := &CmdConfig{}           // blank one
	conf.Load([]byte(configJsonA)) // load configJsonA

	newConf := conf // copy the cmdConfg
	newConf.Servers[1].IsEnabled = true
	if jsonbytes, err := json.Marshal(newConf); err == nil {
		//fmt.Println(string(jsonbytes))
		ioutil.WriteFile("configJsonA.json", []byte(jsonbytes), 0644)
	} else {
		t.Error(err)
	}
	// send a sighup signal to the server
	sigHup()
	time.Sleep(testPauseDuration) // pause for config to reload

	if conn, buffin, err := test.Connect(newConf.Servers[1], 20); err != nil {
		t.Error("Could not connect to new server", newConf.Servers[1].ListenInterface)
	} else {
		if result, err := test.Command(conn, buffin, "HELO example.com"); err == nil {
			expect := "250 enable.test.com Hello"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			}
		} else {
			t.Error(err)
		}
		conn.Close()
	}
	// now disable the server
	newerConf := newConf // copy the cmdConfg
	newerConf.Servers[1].IsEnabled = false
	if jsonbytes, err := json.Marshal(newerConf); err == nil {
		//fmt.Println(string(jsonbytes))
		ioutil.WriteFile("configJsonA.json", []byte(jsonbytes), 0644)
	} else {
		t.Error(err)
	}
	// send a sighup signal to the server
	sigHup()
	time.Sleep(testPauseDuration) // pause for config to reload

	// it should not connect to the server
	if _, _, err := test.Connect(newConf.Servers[1], 20); err == nil {
		t.Error("127.0.0.1:2228 was disabled, but still accepting connections", newConf.Servers[1].ListenInterface)
	}
	// send kill signal and wait for exit
	sigKill()
	serveWG.Wait()

	// did backend started as expected?
	fd, _ := os.Open("./_test/testlog")
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "Server [127.0.0.1:2228] stopped"); i < 0 {
			t.Error("did not stop [127.0.0.1:2228], most likely because Bus.Subscribe(\"server_change:stop_server\" didnt fire")
		}
	}

	// cleanup
	os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonA.json")
	os.Remove("./pidfile.pid")

}

// Start with configJsonD.json,
// then connect to 127.0.0.1:4655 & HELO & try RCPT TO with an invalid host [grr.la]
// then change the config to enable add new host [grr.la] to allowed_hosts
// then write the new config,
// then SIGHUP (to reload config & trigger config update events),
// connect to 127.0.0.1:4655 & HELO & try RCPT TO, grr.la should work

func TestAllowedHostsEvent(t *testing.T) {
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "_test/")
	os.Truncate("./_test/testlog", 0)
	mainlog, _ = log.GetLogger("./_test/testlog", log.DebugLevel.String())
	// start the server by emulating the serve command
	ioutil.WriteFile("configJsonD.json", []byte(configJsonD), 0644)
	conf := &CmdConfig{}           // blank one
	conf.Load([]byte(configJsonD)) // load configJsonD
	cmd := &cobra.Command{}
	configPath = "configJsonD.json"
	var serveWG sync.WaitGroup
	time.Sleep(testPauseDuration)
	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration)

	// now connect and try RCPT TO with an invalid host
	if conn, buffin, err := test.Connect(conf.AppConfig.Servers[1], 20); err != nil {
		t.Error("Could not connect to new server", conf.AppConfig.Servers[1].ListenInterface, err)
	} else {
		if result, err := test.Command(conn, buffin, "HELO test.com"); err == nil {
			expect := "250 secure.test.com Hello"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			} else {
				if result, err = test.Command(conn, buffin, "RCPT TO:<test@grr.la>"); err == nil {
					expect := "454 4.1.1 Error: Relay access denied: grr.la"
					if strings.Index(result, expect) != 0 {
						t.Error("Expected:", expect, "but got:", result)
					}
				}
			}
		}
		conn.Close()
	}

	// now change the config by adding a host to allowed hosts

	newConf := conf // copy the cmdConfg
	newConf.AllowedHosts = append(newConf.AllowedHosts, "grr.la")
	if jsonbytes, err := json.Marshal(newConf); err == nil {
		ioutil.WriteFile("configJsonD.json", []byte(jsonbytes), 0644)
	} else {
		t.Error(err)
	}
	// send a sighup signal to the server to reload config
	sigHup()
	time.Sleep(testPauseDuration) // pause for config to reload

	// now repeat the same conversion, RCPT TO should be accepted
	if conn, buffin, err := test.Connect(conf.AppConfig.Servers[1], 20); err != nil {
		t.Error("Could not connect to new server", conf.AppConfig.Servers[1].ListenInterface, err)
	} else {
		if result, err := test.Command(conn, buffin, "HELO test.com"); err == nil {
			expect := "250 secure.test.com Hello"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			} else {
				if result, err = test.Command(conn, buffin, "RCPT TO:<test@grr.la>"); err == nil {
					expect := "250 2.1.5 OK"
					if strings.Index(result, expect) != 0 {
						t.Error("Expected:", expect, "but got:", result)
					}
				}
			}
		}
		conn.Close()
	}
	// send kill signal and wait for exit
	sigKill()
	serveWG.Wait()
	// did backend started as expected?
	fd, _ := os.Open("./_test/testlog")
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "allowed_hosts config changed, a new list was set"); i < 0 {
			t.Errorf("did not change allowed_hosts, most likely because Bus.Subscribe(\"%s\" didnt fire",
				guerrilla.EventConfigAllowedHosts)
		}
	}
	// cleanup
	//os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonD.json")
	os.Remove("./pidfile.pid")

}

// Test TLS config change event
// start with configJsonD
// should be able to STARTTLS to 127.0.0.1:2525 with no problems
// generate new certs & reload config
// should get a new tls event & able to STARTTLS with no problem

func TestTLSConfigEvent(t *testing.T) {
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "_test/")
	// pause for generated cert to output on slow machines
	time.Sleep(testPauseDuration)
	// did cert output?
	if _, err := os.Stat("./_test/mail2.guerrillamail.com.cert.pem"); err != nil {
		t.Error("Did not create cert ", err)
	}
	mainlog, _ = log.GetLogger("./_test/testlog", log.DebugLevel.String())
	// start the server by emulating the serve command
	ioutil.WriteFile("configJsonD.json", []byte(configJsonD), 0644)
	conf := &CmdConfig{}           // blank one
	conf.Load([]byte(configJsonD)) // load configJsonD
	cmd := &cobra.Command{}
	configPath = "configJsonD.json"
	var serveWG sync.WaitGroup
	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration)

	// Test STARTTLS handshake
	testTlsHandshake := func() {
		if conn, buffin, err := test.Connect(conf.AppConfig.Servers[0], 20); err != nil {
			t.Error("Could not connect to server", conf.AppConfig.Servers[0].ListenInterface, err)
		} else {
			if result, err := test.Command(conn, buffin, "HELO test.com"); err == nil {
				expect := "250 mail.test.com Hello"
				if strings.Index(result, expect) != 0 {
					t.Error("Expected", expect, "but got", result)
				} else {
					if result, err = test.Command(conn, buffin, "STARTTLS"); err == nil {
						expect := "220 2.0.0 Ready to start TLS"
						if strings.Index(result, expect) != 0 {
							t.Error("Expected:", expect, "but got:", result)
						} else {
							tlsConn := tls.Client(conn, &tls.Config{
								InsecureSkipVerify: true,
								ServerName:         "127.0.0.1",
							})
							if err := tlsConn.Handshake(); err != nil {
								t.Error("Failed to handshake", conf.AppConfig.Servers[0].ListenInterface)
							} else {
								conn = tlsConn
								mainlog.Info("TLS Handshake succeeded")
							}

						}
					}
				}
			}
			conn.Close()
		}
	}
	testTlsHandshake()

	if err := os.Remove("./_test/mail2.guerrillamail.com.cert.pem"); err != nil {
		t.Error("could not remove cert", err)
	}
	if err := os.Remove("./_test/mail2.guerrillamail.com.key.pem"); err != nil {
		t.Error("could not remove key", err)
	}

	// generate a new cert
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "_test/")
	// pause for generated cert to output
	time.Sleep(testPauseDuration)
	// did cert output?
	if _, err := os.Stat("./_test/mail2.guerrillamail.com.cert.pem"); err != nil {
		t.Error("Did not create cert ", err)
	}

	sigHup()

	time.Sleep(testPauseDuration * 2) // pause for config to reload
	testTlsHandshake()

	//time.Sleep(testPauseDuration)
	// send kill signal and wait for exit
	sigKill()
	serveWG.Wait()
	// did backend started as expected?
	fd, _ := os.Open("./_test/testlog")
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "Server [127.0.0.1:2552] new TLS configuration loaded"); i < 0 {
			t.Error("did not change tls, most likely because Bus.Subscribe(\"server_change:tls_config\" didnt fire")
		}
	}

	// cleanup
	os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonD.json")
	os.Remove("./pidfile.pid")

}

// Testing starting a server with a bad TLS config
// It should not start, return exit code 1
func TestBadTLSStart(t *testing.T) {
	// Need to run the test in a different process by executing a command
	// because the serve() does os.Exit when starting with a bad TLS config
	if os.Getenv("BE_CRASHER") == "1" {
		// do the test
		// first, remove the good certs, if any
		if err := os.Remove("./_test/mail2.guerrillamail.com.cert.pem"); err != nil {
			mainlog.WithError(err).Error("could not remove ./_test/mail2.guerrillamail.com.cert.pem")
		} else {
			mainlog.Info("removed ./_test/mail2.guerrillamail.com.cert.pem")
		}
		// next run the server
		ioutil.WriteFile("configJsonD.json", []byte(configJsonD), 0644)
		conf := &CmdConfig{}           // blank one
		conf.Load([]byte(configJsonD)) // load configJsonD

		cmd := &cobra.Command{}
		configPath = "configJsonD.json"
		var serveWG sync.WaitGroup

		serveWG.Add(1)
		go func() {
			Start(cmd, []string{})
			serveWG.Done()
		}()
		time.Sleep(testPauseDuration)

		sigKill()
		serveWG.Wait()

		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestBadTLSStart")
	cmd.Env = append(os.Environ(), "BE_CRASHER=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Error("Server started with a bad TLS config, was expecting exit status 1")
	// cleanup
	os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonD.json")
	os.Remove("./pidfile.pid")
}

// Test config reload with a bad TLS config
// It should ignore the config reload, keep running with old settings
func TestBadTLSReload(t *testing.T) {
	mainlog, _ = log.GetLogger("./_test/testlog", log.DebugLevel.String())
	// start with a good vert
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "./_test/")
	// start the server by emulating the serve command
	ioutil.WriteFile("configJsonD.json", []byte(configJsonD), 0644)
	conf := &CmdConfig{}           // blank one
	conf.Load([]byte(configJsonD)) // load configJsonD
	cmd := &cobra.Command{}
	configPath = "configJsonD.json"
	var serveWG sync.WaitGroup

	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration)

	if conn, buffin, err := test.Connect(conf.AppConfig.Servers[0], 20); err != nil {
		t.Error("Could not connect to server", conf.AppConfig.Servers[0].ListenInterface, err)
	} else {
		if result, err := test.Command(conn, buffin, "HELO test.com"); err == nil {
			expect := "250 mail.test.com Hello"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			}
		}
	}
	// write some trash data
	ioutil.WriteFile("./_test/mail2.guerrillamail.com.cert.pem", []byte("trash data"), 0664)
	ioutil.WriteFile("./_test/mail2.guerrillamail.com.key.pem", []byte("trash data"), 0664)

	newConf := conf // copy the cmdConfg

	if jsonbytes, err := json.Marshal(newConf); err == nil {
		ioutil.WriteFile("configJsonD.json", []byte(jsonbytes), 0644)
	} else {
		t.Error(err)
	}
	// send a sighup signal to the server to reload config
	sigHup()
	time.Sleep(testPauseDuration) // pause for config to reload

	// we should still be able to to talk to it

	if conn, buffin, err := test.Connect(conf.AppConfig.Servers[0], 20); err != nil {
		t.Error("Could not connect to server", conf.AppConfig.Servers[0].ListenInterface, err)
	} else {
		if result, err := test.Command(conn, buffin, "HELO test.com"); err == nil {
			expect := "250 mail.test.com Hello"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			}
		}
	}

	sigKill()
	serveWG.Wait()

	// did config reload fail as expected?
	fd, _ := os.Open("./_test/testlog")
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "cannot use TLS config for"); i < 0 {
			t.Error("[127.0.0.1:2552] did not reject our tls config as expected")
		}
	}
	// cleanup
	os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonD.json")
	os.Remove("./pidfile.pid")
}

// Test for when the server config Timeout value changes
// Start with configJsonD.json

func TestSetTimeoutEvent(t *testing.T) {
	mainlog, _ = log.GetLogger("./_test/testlog", log.DebugLevel.String())
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "./_test/")
	// start the server by emulating the serve command
	ioutil.WriteFile("configJsonD.json", []byte(configJsonD), 0644)
	conf := &CmdConfig{}           // blank one
	conf.Load([]byte(configJsonD)) // load configJsonD
	cmd := &cobra.Command{}
	configPath = "configJsonD.json"
	var serveWG sync.WaitGroup

	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration)

	// set the timeout to 1 second

	newConf := conf // copy
	newConf.Servers[0].Timeout = 1
	if jsonbytes, err := json.Marshal(newConf); err == nil {
		ioutil.WriteFile("configJsonD.json", []byte(jsonbytes), 0644)
	} else {
		t.Error(err)
	}

	// send a sighup signal to the server to reload config
	sigHup()
	time.Sleep(testPauseDuration) // config reload

	var waitTimeout sync.WaitGroup
	if conn, buffin, err := test.Connect(conf.AppConfig.Servers[0], 20); err != nil {
		t.Error("Could not connect to server", conf.AppConfig.Servers[0].ListenInterface, err)
	} else {
		waitTimeout.Add(1)
		go func() {
			if result, err := test.Command(conn, buffin, "HELO test.com"); err == nil {
				expect := "250 mail.test.com Hello"
				if strings.Index(result, expect) != 0 {
					t.Error("Expected", expect, "but got", result)
				} else {
					b := make([]byte, 1024)
					conn.Read(b)
				}
			}
			waitTimeout.Done()
		}()
	}

	// wait for timeout
	waitTimeout.Wait()

	// so the connection we have opened should timeout by now

	// send kill signal and wait for exit
	sigKill()
	serveWG.Wait()
	// did backend started as expected?
	fd, _ := os.Open("./_test/testlog")
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "i/o timeout"); i < 0 {
			t.Error("Connection to 127.0.0.1:2552 didn't timeout as expected")
		}
	}
	// cleanup
	os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonD.json")
	os.Remove("./pidfile.pid")

}

// Test debug level config change
// Start in log_level = debug
// Load config & start server
func TestDebugLevelChange(t *testing.T) {
	//mainlog, _ = log.GetLogger("./_test/testlog")
	testcert.GenerateCert("mail2.guerrillamail.com", "", 365*24*time.Hour, false, 2048, "P256", "./_test/")
	// start the server by emulating the serve command
	ioutil.WriteFile("configJsonD.json", []byte(configJsonD), 0644)
	conf := &CmdConfig{}           // blank one
	conf.Load([]byte(configJsonD)) // load configJsonD
	conf.LogLevel = "debug"
	cmd := &cobra.Command{}
	configPath = "configJsonD.json"
	var serveWG sync.WaitGroup

	serveWG.Add(1)
	go func() {
		Start(cmd, []string{})
		serveWG.Done()
	}()
	time.Sleep(testPauseDuration)

	if conn, buffin, err := test.Connect(conf.AppConfig.Servers[0], 20); err != nil {
		t.Error("Could not connect to server", conf.AppConfig.Servers[0].ListenInterface, err)
	} else {
		if result, err := test.Command(conn, buffin, "HELO test.com"); err == nil {
			expect := "250 mail.test.com Hello"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			}
		}
		conn.Close()
	}
	// set the log_level to info

	newConf := conf // copy the cmdConfg
	newConf.LogLevel = "info"
	if jsonbytes, err := json.Marshal(newConf); err == nil {
		ioutil.WriteFile("configJsonD.json", []byte(jsonbytes), 0644)
	} else {
		t.Error(err)
	}
	// send a sighup signal to the server to reload config
	sigHup()
	time.Sleep(testPauseDuration) // log to change

	// connect again, this time we should see info
	if conn, buffin, err := test.Connect(conf.AppConfig.Servers[0], 20); err != nil {
		t.Error("Could not connect to server", conf.AppConfig.Servers[0].ListenInterface, err)
	} else {
		if result, err := test.Command(conn, buffin, "NOOP"); err == nil {
			expect := "200 2.0.0 OK"
			if strings.Index(result, expect) != 0 {
				t.Error("Expected", expect, "but got", result)
			}
		}
		conn.Close()
	}

	// send kill signal and wait for exit
	sigKill()
	serveWG.Wait()
	// did backend started as expected?
	fd, _ := os.Open("./_test/testlog")
	if read, err := ioutil.ReadAll(fd); err == nil {
		logOutput := string(read)
		//fmt.Println(logOutput)
		if i := strings.Index(logOutput, "log level changed to [info]"); i < 0 {
			t.Error("Log level did not change to [info]")
		}
		// This should not be there:
		if i := strings.Index(logOutput, "Client sent: NOOP"); i != -1 {
			t.Error("Log level did not change to [info], we are still seeing debug messages")
		}
	}
	// cleanup
	os.Truncate("./_test/testlog", 0)
	os.Remove("configJsonD.json")
	os.Remove("./pidfile.pid")

}

func TestMailDirDelivery(t *testing.T) {
	ioutil.WriteFile("./_test/config.json", []byte(configJsonE), 0644)
	d := guerrilla.Daemon{}
	d.AddProcessor("MailDir", maildir_processor.Processor)

	_, err := d.LoadConfig("./_test/config.json")
	if err != nil {
		t.Error("could not read config:", err)
		return
	}
	err = d.Start()
	if err != nil {
		t.Error("could not start daemon:", err)
		return
	}

	conn, err := net.Dial("tcp", "127.0.0.1:3535")
	if err != nil {
		// handle error
		t.Error("cannot connect", err)
		return
	}
	in := bufio.NewReader(conn)
	str, err := in.ReadString('\n')
	fmt.Println(str)
	fmt.Fprint(conn, "HELO test.com\r\n")
	str, err = in.ReadString('\n')
	fmt.Println("[HELO] " + str)
	fmt.Fprint(conn, "MAIL FROM:<test@example.com>r\r\n")
	str, err = in.ReadString('\n')
	fmt.Println("[MAIL] " + str)
	fmt.Fprint(conn, "RCPT TO:<test@grr.la>\r\n")
	str, err = in.ReadString('\n')
	fmt.Println("[RCPT] " + str)
	fmt.Fprint(conn, "DATA\r\n")
	str, err = in.ReadString('\n')
	fmt.Println("[DATA] " + str)
	fmt.Fprint(conn, "Subject: Test subject\r\n")
	fmt.Fprint(conn, "\r\n")
	fmt.Fprint(conn, "A an email body\r\n")
	fmt.Fprint(conn, ".\r\n")
	str, err = in.ReadString('\n')
	fmt.Println(str)

	_, err = os.Stat("./_test/Maildir/new")
	if err != nil {
		t.Error("cannot confirm the existance of _test/Maildir/new ", err)
		return
	}
	if empty, err := isEmpty("./_test/Maildir/new"); empty || err != nil {
		t.Error("looks like no email was delivered, _test/Maildir/new looks empty")
	}
	// clean up
	os.RemoveAll("./_test/Maildir/new")

}

func isEmpty(name string) (bool, error) {
	f, err := os.Open(name)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1) // Or f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err // Either not empty or error, suits both cases
}
