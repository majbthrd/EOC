cmd/app/
- removed running in the background (including tray and pretending to stop running)
- removed "hidden" mode
- removed log files and log file rotation
- removed deference to other instances of software
- removed download of arbitrary upgrade code from the cloud
- removed "tools" that route web traffic through Ollama cloud servers
- removed code that performs user logins to Ollama cloud servers

store/
- wholesale replacement of mySQL database with ephemeral, RAM-based database

tools/
- removed web traffic through Ollama cloud servers

ui/app/src/
- defeated options to route web traffic through Ollama cloud servers
- defeated polling Ollama web servers for alternate LLMs
- removed chat history and settings GUI controls
- removed some unused clutter

ui/
- remove updater (download of arbitrary upgrade code)
- simplified reverse proxy that shims into actual LLM
- removed code that interfaces to Ollama cloud
- removed code that would open a cloud-provided arbitrary web page
- removed "tools" that route web traffic through Ollama cloud servers
- removed controls for now defeated Chat History
- removed download of settings from Ollama cloud

