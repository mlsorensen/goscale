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
2. the `cmd/mockscale/example.go` demonstrates how to use a MOCK implementation of scale in a real program.
3. the `cmd/scanner/scan.go` should scan for any currently active, supported scales and print them via
   ``` go run cmd/scanner/scan.go```

## Current Status

This is an early implementation with a mock backend. The current implementation:

- Provides basic interface definition
- Includes a simple implementation that simulates weight updates
- Supports core scale operations

## Next Steps

1. Implement specific scales

Feel free to contribute to the implementation!

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/your-feature-name`)
3. Commit your changes (`git commit -m "Add your feature"`)
4. Push to the branch (`git push origin feature/your-feature-name`)
5. Open a Pull Request

## License

[Your License Here]
