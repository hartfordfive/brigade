package main

type HttpDirectives struct {
	Directives []HttpDirective `json:"url_list"`
	MinDelay   int             `json:"min_delay"`
	MaxDelay   int             `json:"max_delay"`
	Proxy      string          `json:"proxy,omitempty"`
}

type HttpDirective struct {
	Url      string       `json:"url"`
	Method   string       `json:"method"`
	PostData string       `json:"data"`
	Headers  []HttpHeader `json:"headers"`
	Cookies  []Cookie     `json:"coookies"`
	Weight   int          `json:"weight"`
	Proxy    string       `json:"proxy,omitempty"`
}

type HttpHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type Cookie struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SshDirectives struct {
	Directives []SshDirective `json:"ssh_actions"`
}

type SshDirective struct {
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
	UseRndPass bool   `json:"use_rnd_pass"`
	Password   string `json:"password"`
	Command    string `json:"command"`
	Repeat     int    `json:"repeat"`
	Weight     int    `json:"weight"`
}

type ExecDirective struct {
	Command string `json:"command"`
	Repeat  int    `json:"repeat"`
}

type ScriptDirective struct {
	ScriptName  string `json:"script_name,omitempty"`
	ScriptType  string `json:"script_type,omitempty"`
	ScriptUrl   string `json:"script_url,omitempty"`
	ScriptBody  string `json:"script_body,omitempty"`
	RepeatMode  string `json:"repeat_mode"`
	RepeatTimes int    `json:"repeat_times"`
}
