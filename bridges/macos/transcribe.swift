import AVFoundation
import Foundation
import Speech

struct TranscribeJob: Codable {
    let inputPath: String
    let outputPath: String
    let locale: String?
}

struct TranscribeRequest: Codable {
    let jobs: [TranscribeJob]
}

struct TranscribeResult: Codable {
    let inputPath: String
    let outputPath: String
    let text: String
    let error: String?
}

struct TranscribeResponse: Codable {
    let results: [TranscribeResult]
}

enum BridgeError: Error, LocalizedError {
    case unsupportedOS
    case unsupportedLocale(String)
    case transcriberUnavailable

    var errorDescription: String? {
        switch self {
        case .unsupportedOS:
            return "Apple Speech transcription requires macOS 26 or later"
        case let .unsupportedLocale(locale):
            return "Unsupported speech locale: \(locale)"
        case .transcriberUnavailable:
            return "SpeechTranscriber is not available on this device"
        }
    }
}

@available(macOS 26.0, *)
func transcribe(job: TranscribeJob) async throws -> TranscribeResult {
    let inputURL = URL(fileURLWithPath: job.inputPath)
    let outputURL = URL(fileURLWithPath: job.outputPath)
    let locale = Locale(identifier: job.locale?.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty == false ? job.locale! : Locale.current.identifier)

    guard FileManager.default.fileExists(atPath: inputURL.path) else {
        throw CocoaError(.fileNoSuchFile)
    }
    guard SpeechTranscriber.isAvailable else {
        throw BridgeError.transcriberUnavailable
    }

    let supportedLocales = await SpeechTranscriber.supportedLocales
    guard supportedLocales.contains(where: { $0.identifier == locale.identifier || $0.identifier(.bcp47) == locale.identifier(.bcp47) }) else {
        throw BridgeError.unsupportedLocale(locale.identifier)
    }

    for reservedLocale in await AssetInventory.reservedLocales {
        await AssetInventory.release(reservedLocale: reservedLocale)
    }

    let transcriber = SpeechTranscriber(
        locale: locale,
        transcriptionOptions: [],
        reportingOptions: [],
        attributeOptions: []
    )
    let modules: [any SpeechModule] = [transcriber]

    let installedLocales = await SpeechTranscriber.installedLocales
    if !installedLocales.contains(where: { $0.identifier == locale.identifier || $0.identifier(.bcp47) == locale.identifier(.bcp47) }) {
        if let request = try await AssetInventory.assetInstallationRequest(supporting: modules) {
            try await request.downloadAndInstall()
        }
    }

    let analyzer = SpeechAnalyzer(modules: modules)
    let audioFile = try AVAudioFile(forReading: inputURL)
    try await analyzer.start(inputAudioFile: audioFile, finishAfterFile: true)

    var transcript = AttributedString("")
    for try await result in transcriber.results {
        transcript += result.text
    }

    let text = String(transcript.characters)
    try FileManager.default.createDirectory(at: outputURL.deletingLastPathComponent(), withIntermediateDirectories: true)
    try text.write(to: outputURL, atomically: true, encoding: .utf8)
    return TranscribeResult(inputPath: job.inputPath, outputPath: job.outputPath, text: text, error: nil)
}

let input = FileHandle.standardInput.readDataToEndOfFile()
let request = try JSONDecoder().decode(TranscribeRequest.self, from: input)

let semaphore = DispatchSemaphore(value: 0)
var response: TranscribeResponse?

Task {
    var results: [TranscribeResult] = []
    for job in request.jobs {
        do {
            if #available(macOS 26.0, *) {
                results.append(try await transcribe(job: job))
            } else {
                throw BridgeError.unsupportedOS
            }
        } catch {
            results.append(TranscribeResult(
                inputPath: job.inputPath,
                outputPath: job.outputPath,
                text: "",
                error: error.localizedDescription
            ))
        }
    }
    response = TranscribeResponse(results: results)
    semaphore.signal()
}

semaphore.wait()
let output = try JSONEncoder().encode(response ?? TranscribeResponse(results: []))
FileHandle.standardOutput.write(output)
