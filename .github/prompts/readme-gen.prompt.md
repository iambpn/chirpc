---
mode: agent
tools: ['edit', 'runNotebooks', 'search', 'new', 'runCommands', 'runTasks', 'usages', 'vscodeAPI', 'think', 'problems', 'changes', 'testFailure', 'openSimpleBrowser', 'fetch', 'githubRepo', 'extensions', 'todos', 'runTests']
---

You are an expert software developer. Your task is to generate a comprehensive README.md file for a GitHub repository based on the provided context. The README should include sections such as Introduction, Features, Installation, Usage, Contributing, and License. Ensure that the content is clear, concise, and well-structured to help users understand the purpose and functionality of the project.

Review all the code from the codebase and generate the README.md file accordingly.

For generating the usage section, consider the common use case and see the ./cmd/js/ example for reference. Yo need to use the specific npm package "ts-axios-wrapper" to use the generated typescript types and api client.

If You are confused about any part of the codebase, ask for clarification before proceeding.

Readme structure to follow:

- Header: with project name and short description
- Introduction: Brief overview of the project
- Features: List of key features
- Installation: Step-by-step instructions on how to install the project
- Usage: Detailed guide on how to use the project, including code examples
- Exposed APIs: Description of the main APIs provided by the project
- Contributing: Guidelines for contributing to the project
- License: Information about the project's license
