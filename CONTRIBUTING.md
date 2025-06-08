# Contributing to OVN Control Platform

Thank you for your interest in contributing to OVN Control Platform! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [How to Contribute](#how-to-contribute)
- [Coding Standards](#coding-standards)
- [Testing Guidelines](#testing-guidelines)
- [Documentation](#documentation)
- [Submitting Changes](#submitting-changes)
- [Review Process](#review-process)

## Code of Conduct

By participating in this project, you agree to abide by our [Code of Conduct](CODE_OF_CONDUCT.md). Please read it before contributing.

## Getting Started

### Prerequisites

Before you begin, ensure you have:

- Go 1.22 or higher
- Node.js 20 or higher
- Docker and Docker Compose
- PostgreSQL 14 or higher
- Git
- A GitHub account

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:
   ```bash
   git clone https://github.com/YOUR_USERNAME/ovncp.git
   cd ovncp
   ```
3. Add the upstream remote:
   ```bash
   git remote add upstream https://github.com/lspecian/ovncp.git
   ```

## Development Setup

### Backend Setup

```bash
# Install Go dependencies
go mod download

# Install development tools
make install-tools

# Set up local database
createdb ovncp_dev
make migrate

# Copy environment configuration
cp .env.example .env
# Edit .env with your local configuration

# Run the backend
make run-api
```

### Frontend Setup

```bash
cd web

# Install dependencies
npm install

# Run development server
npm run dev
```

### Running with Docker Compose

```bash
# Start all services
docker-compose up

# Run in detached mode
docker-compose up -d

# View logs
docker-compose logs -f
```

## How to Contribute

### Finding Issues

- Check our [issue tracker](https://github.com/lspecian/ovncp/issues) for open issues
- Look for issues labeled `good first issue` or `help wanted`
- Feel free to ask questions on issues you'd like to work on

### Creating Issues

When creating an issue, please:

1. Use a clear and descriptive title
2. Provide a detailed description of the issue
3. Include steps to reproduce (for bugs)
4. Add relevant labels
5. Include system information if relevant

### Feature Requests

For feature requests:

1. Check if the feature has already been requested
2. Clearly explain the use case
3. Describe the expected behavior
4. Provide examples if possible

## Coding Standards

### Go Code Style

- Follow [Effective Go](https://golang.org/doc/effective_go.html) guidelines
- Use `gofmt` to format your code
- Use `golangci-lint` for linting:
  ```bash
  make lint
  ```
- Write idiomatic Go code
- Add comments for exported functions and types

Example:
```go
// LogicalSwitch represents an OVN logical switch
type LogicalSwitch struct {
    UUID        string    `json:"uuid"`
    Name        string    `json:"name"`
    Description string    `json:"description,omitempty"`
    CreatedAt   time.Time `json:"created_at"`
}

// NewLogicalSwitch creates a new logical switch with the given name
func NewLogicalSwitch(name string) (*LogicalSwitch, error) {
    if name == "" {
        return nil, errors.New("switch name cannot be empty")
    }
    
    return &LogicalSwitch{
        UUID:      uuid.New().String(),
        Name:      name,
        CreatedAt: time.Now(),
    }, nil
}
```

### JavaScript/TypeScript Code Style

- Use ESLint and Prettier configurations provided
- Follow React best practices
- Use TypeScript for type safety
- Write functional components with hooks

Example:
```typescript
interface SwitchListProps {
  switches: LogicalSwitch[];
  onDelete: (id: string) => void;
}

export const SwitchList: React.FC<SwitchListProps> = ({ switches, onDelete }) => {
  const handleDelete = useCallback((id: string) => {
    if (confirm('Are you sure you want to delete this switch?')) {
      onDelete(id);
    }
  }, [onDelete]);

  return (
    <div className="switch-list">
      {switches.map((switch) => (
        <SwitchCard
          key={switch.uuid}
          switch={switch}
          onDelete={handleDelete}
        />
      ))}
    </div>
  );
};
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

<body>

<footer>
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements

Examples:
```bash
feat(api): add pagination support for switches endpoint

fix(web): resolve race condition in topology view

docs(readme): update installation instructions
```

## Testing Guidelines

### Writing Tests

#### Go Tests

```go
func TestCreateLogicalSwitch(t *testing.T) {
    tests := []struct {
        name    string
        input   CreateSwitchRequest
        want    *LogicalSwitch
        wantErr bool
    }{
        {
            name: "valid switch",
            input: CreateSwitchRequest{
                Name: "test-switch",
            },
            want: &LogicalSwitch{
                Name: "test-switch",
            },
            wantErr: false,
        },
        {
            name: "empty name",
            input: CreateSwitchRequest{
                Name: "",
            },
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := CreateLogicalSwitch(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateLogicalSwitch() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("CreateLogicalSwitch() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

#### JavaScript/TypeScript Tests

```typescript
describe('SwitchList', () => {
  const mockSwitches: LogicalSwitch[] = [
    { uuid: '1', name: 'switch1', createdAt: new Date() },
    { uuid: '2', name: 'switch2', createdAt: new Date() },
  ];

  it('renders all switches', () => {
    const { getAllByTestId } = render(
      <SwitchList switches={mockSwitches} onDelete={jest.fn()} />
    );
    
    expect(getAllByTestId('switch-card')).toHaveLength(2);
  });

  it('calls onDelete when delete button clicked', () => {
    const onDelete = jest.fn();
    const { getByTestId } = render(
      <SwitchList switches={mockSwitches} onDelete={onDelete} />
    );
    
    fireEvent.click(getByTestId('delete-switch-1'));
    expect(onDelete).toHaveBeenCalledWith('1');
  });
});
```

### Running Tests

```bash
# Run all tests
make test

# Run Go tests with coverage
make test-go-coverage

# Run JavaScript tests
make test-js

# Run integration tests
make test-integration

# Run E2E tests
make test-e2e
```

### Test Coverage

- Aim for at least 80% test coverage
- Critical paths should have 100% coverage
- Use `make test-coverage` to check coverage

## Documentation

### Code Documentation

- Document all exported functions, types, and constants
- Use clear, concise language
- Include examples where helpful
- Keep documentation up-to-date with code changes

### API Documentation

- Update OpenAPI specification for API changes
- Include request/response examples
- Document error responses
- Add security requirements

### User Documentation

- Update user guides for new features
- Include screenshots where appropriate
- Test documentation steps
- Keep language clear and accessible

## Submitting Changes

### Pull Request Process

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes and commit:
   ```bash
   git add .
   git commit -m "feat(scope): add new feature"
   ```

3. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

4. Create a Pull Request on GitHub

### Pull Request Template

```markdown
## Description
Brief description of the changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] No new warnings
```

### PR Guidelines

- Keep PRs focused and small
- Link related issues
- Update documentation
- Add tests for new functionality
- Ensure CI passes
- Respond to review feedback promptly

## Review Process

### What to Expect

1. Automated CI checks will run
2. A maintainer will review your PR
3. You may receive feedback or requests for changes
4. Once approved, your PR will be merged

### Review Criteria

- Code quality and style
- Test coverage
- Documentation
- Performance impact
- Security considerations
- Backward compatibility

### Getting Help

If you need help:

- Comment on the issue or PR
- Join our [Slack channel](https://ovncp.slack.com)
- Attend our [community meetings](https://github.com/lspecian/ovncp/wiki/Community-Meetings)

## Recognition

Contributors will be:
- Listed in our [CONTRIBUTORS](CONTRIBUTORS.md) file
- Mentioned in release notes
- Eligible for contributor badges

## Thank You!

Your contributions make OVN Control Platform better for everyone. We appreciate your time and effort!

## Additional Resources

- [Development Guide](docs/development.md)
- [Architecture Overview](docs/architecture.md)
- [API Documentation](https://api.ovncp.io/docs)
- [Community Forum](https://forum.ovncp.io)