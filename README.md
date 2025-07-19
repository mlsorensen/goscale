# Go Bluetooth Scale Interface

A Go implementation of a Bluetooth scale interface with weight tracking capabilities.

## Features

- Asynchronous weight updates through channels
- Tare functionality (with blocking support)
- Sleep timeout configuration
- Battery charge monitoring
- Clean interface-based design for easy implementation swapping

## Getting Started
1. clone the repository
2. the cmd/example.go should be runnable and demonstrate how to use a MOCK implementation of scale

## Current Status

This is an early implementation with a dummy backend. The current implementation:

- Provides basic interface definition
- Includes a simple implementation that simulates weight updates
- Supports core scale operations

## Next Steps

1. Implement platform-specific Bluetooth connectivity:
   - Windows: Use Windows Bluetooth API
   - macOS/Linux: Use BlueZ or equivalent

2. Add error handling and retry mechanisms
3. Implement proper connection management
4. Add more sophisticated parsing of actual scale data

Feel free to contribute to the implementation!

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature-name`)
3. Commit your changes (`git commit -m "Add your feature"`)
4. Push to the branch (`git push origin feature/your-feature-name`)
5. Open a Pull Request

## License

[Your License Here]
