import Foundation
import SpeechCore

let input = FileHandle.standardInput.readDataToEndOfFile()
FileHandle.standardOutput.write(try await runTranscriptBridge(input: input))
