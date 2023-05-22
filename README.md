# go-auto-gpt

**go-auto-gpt** is influenced from the experiment done with [auto-gpt](https://github.com/Significant-Gravitas/Auto-GPT), an open-source project that showcases the capabilities of GPT language models to autonomously solve a goal.

Instead of copying the implementation, I've used this project a proof of concept to quickly try to both learn and bring a new design to how to solve autonomous tasks. Specifically the orchestration of agents.

## What I Learned
  - It will become increasingly more important to have increased context window size from models to build more complex autonomous agents with lots of text
  - The need for DSLs for creating optimized prompts to limit tokens will be useful (see: [We need new DSLs for the era of LLMs](https://zainhoda.github.io/2023/05/20/dsls-for-llms.html))
  - The orchestration of agents is important and ability to communicate in a cluster might let agents delegate work when they need something
  - Long-term memory will be very useful for not repeating the same observed problems
  - Agents with custom or specialized LLMs will be useful
  - How well you construct a chain can help solve a task better by providing more context to solving the problem
  - DAG's to dynamically construct chains will be useful
  - It can get expensive quickly, even for simple tasks

## Some Existing Features
- an HTTP server with an API
  - can create new goals to be solved
  - goals are tracked by an id and can be queried
- the use of actors as a framework for building agents (see: [actor model](https://en.wikipedia.org/wiki/Actor_model)), through the use of [protoactor-go](https://github.com/asynkron/protoactor-go), actors can be used to:
  - run tasks both asynchronously or synchronously of each other
  - run remotely of each other and communicate through gRPC for network transport
  - be written in different languages and communicate through message contracts (cross-platform)
  - spawn new actors to breakdown tasks through the use of a supervisor
  - broadcast a task to a cluster of actors

## Agents
  - Planner: takes a goal from a user and breaks it down into a plan of tasks
  - Supervisor: manages the queue of tasks and delegation of tasks to other Agents
  - Terminal: has the ability to run commands and diagnose why commands fail to run then retry
  - Search: todo

## Current Limitations
- Lacking proper chains and memory
- No persistent memory
- No embeddings
- Only setup to run text-davinci-003 (this can be switched in the code)
- Only one tool (it will get confused if you ask something it can't do, e.g. I asked it to search for trends in AI and it tried to search the filesystem)
- Terminal agent will sometimes try to brute force its way to a solution
  - because of this, it has a hardcoded maxAttempts for diagnosing problems

## Usage

You will need to set `OPENAI_API_KEY` to your [key](https://platform.openai.com/account/api-keys).

### Warning :exclamation:
The agents have the ability to execute arbitrary code on your machine! It is recommended to use the [sandbox.Dockerfile](sandbox.Dockerfile).

If you want to view any output, you'll want to mount a volume to the container `-v ./sandbox:/app/sandbox`.

To create a new goal to be solved by the agents, simply make a request to the API:
```bash
curl --location --request POST 'localhost:8080/new' \
--header 'Content-Type: application/json' \
--data '{
    "goal": "write a python file that prints hello world and execute the file"
}'
```

Then periodically check the status:
```bash
curl --location --request GET 'localhost:8080/status/$ID'
```

When the goal has been completed, the state will change to `finished` and you'll be able to review the full history and state from each task, including chat results from the LLM.

## Todo
Nice to haves if I continue this project.
- [ ] pass in config to change consts
- [ ] ask for help from the user if a task fails
- [ ] update to use `langchaingo` for chains and memory (when available or alternative library)
- [ ] create embeddings
- [ ] add persistent vectorstore
- [ ] add persistent memory
- [ ] remote actors
- [ ] add search tool
- [ ] clean up the quickly written code