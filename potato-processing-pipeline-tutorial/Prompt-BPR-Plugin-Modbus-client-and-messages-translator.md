Develop a BPR (Bindu Plugin/Processor) in Go that acts as a universal Modbus Client. 
Requirements:
1. It must accept a configuration message containing the necessary connection details (IP, port, timeouts) to connect to the Modbus server.
2. It must process semantic message types and translate them into Modbus commands. 
3. Generalize the design by implementing two core semantic message types: 
   - `ModbusSet`: The plugin receives this message and transforms its payload into a Modbus write command to the server.
   - `ModbusGet`: The plugin receives this, performs a Modbus read request, and transforms the returned data back into a semantic message type to be published to the broker.
The design must be modular and reusable so it can communicate with PLCs in washing machines, production lines, and other industrial devices.