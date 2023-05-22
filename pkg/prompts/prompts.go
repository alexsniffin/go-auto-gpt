package prompts

// todo sanitize responses from the structure provided by a prompt
var (
	PlanTemplate = `
You are an intelligent AI who specializes in planning. As part of a plan to solve a goal: "{{.Goal}}", 
devise a plan of tasks to execute on how to solve this goal.

Each task should be solved independently of one another and any resources should be assumed to be stored in the directory ./tmp 
which can be used between tasks.

Tasks are costly, so try to use as few tasks as possible to complete the goal. 

Try to solve simple goals with only one task.

Limit the retrieval of resources and computation time when possible.

Provide your response in the following json format, where the field tasks is an array of strings:
{
    "tasks": [{LIST_OF_REQUIREMENTS}],
}
`

	TaskTemplate = `
You are an intelligent AI who specializes in solving tasks on a computer. As part of a plan to solve a goal: {{.Goal}}

Here is an ordered json list of steps I have done so far to solve this goal: 
"{{.History}}"

Any resources from previous steps should be assumed to be stored in the directory ./tmp.

I have been given a new task to complete for this goal: "{{.Task}}"

Find the the best way to complete the task using only one tool from only the following list:
	- TERMINAL
		- preference: use verbose flags where possible and avoid any dangerous commands
		- description: a bash based unix terminal
		- interface: Input([Command: string]): Output(OutputFile: []File)
	- ROCKET_SHIP

Pick one tool to complete the task.

Given the tool you choose, provide a value for each argument to the input. Each input should be cast to a string literal.

Provide feedback on your reasoning, give any limitations and provide the expected outcome.

Fill in the following json format, escape any invalid characters in the values, return only what is in the json block, e.g. {}:
{
    "tool": "{YOUR_DESIRED_TOOL}",
	"inputs": ["{ARRAY_OF_INPUTS}"],
    "reasoning": "{YOUR_REASONING}",
    "limitations": "{YOUR_LIMITATIONS}"
    "outcome": "{EXPECTED_OUTCOME}"
}
` // todo remove history when langchaingo supports it

	//	- SEARCH
	//		- preference: be concise and use keywords
	//		- description: a search engine (e.g. Google)
	//		- interface: Input([Query: string]): Output(Results: []string)
	CommandDiagnoseTemplate = `
You are an intelligent AI who specializes using a bash terminal, your OS is Debian and here is a non-exhaustive list of commands you might have access to:
[
    "ls", "cd", "pwd", "cp", "mv", "rm", "mkdir", "cat", "grep",
    "sed", "awk", "gzip", "gunzip", "tar", "zip", "unzip", "nano",
    "vim", "apt-get", "apt-cache", "dpkg", "ping", "ifconfig",
    "netstat", "traceroute", "nslookup", "wget", "curl", "ps",
    "top", "kill", "killall", "pgrep", "pkill", "cut", "sort",
    "uniq", "head", "tail", "wc", "tee", "tr"
]

You're trying to solve the following task: {{.Task}}

Here is a history of the commands that you've executed so far, in a json list: 
{{.PreviousAttempts}}

The field "command" is the command you tried, "error" is any error from the command, "reason" is why you ran it.

Your previous commands didn't help you solve the first command in the list.

Review the previous commands and determine a new command that will let you run the first command.

Do not repeat yourself, do not try the last command.

A missing command "command not found" means that you need to install the command. Provide the right flag like -Y or -y to install the command so you aren't prompted.

Don't use sudo.

Provide your next command in the following json format:
{
    "command": "{NEW_COMMAND}",
	"reason": "{REASON}"
}
`

	// todo make the list of commands a prompt.. let the agent use its memory and reasoning to determine what it should do
)
