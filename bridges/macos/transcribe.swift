import AVFoundation
import Foundation
import Speech

struct TranscribeJob: Codable {
    let inputPath: String
    let outputPath: String
    let locale: String?
    let contextualStrings: [String]?
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
    let localeIdentifier = job.locale?.trimmingCharacters(in: .whitespacesAndNewlines)
    let locale = Locale(identifier: localeIdentifier?.isEmpty == false ? localeIdentifier! : "ko-KR")

    guard FileManager.default.fileExists(atPath: inputURL.path) else {
        throw CocoaError(.fileNoSuchFile)
    }
    guard SpeechTranscriber.isAvailable else {
        throw BridgeError.transcriberUnavailable
    }

    let requestedLocale = locale.identifier(.bcp47)
    let supportedLocales = await SpeechTranscriber.supportedLocales
    guard supportedLocales.contains(where: { $0.identifier(.bcp47) == requestedLocale }) else {
        throw BridgeError.unsupportedLocale(requestedLocale)
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
    if !installedLocales.contains(where: { $0.identifier(.bcp47) == requestedLocale }) {
        if let request = try await AssetInventory.assetInstallationRequest(supporting: modules) {
            try await request.downloadAndInstall()
        }
    }

    let analyzer = SpeechAnalyzer(modules: modules)
    let analysisContext = AnalysisContext()
    let contextualStrings = Array(Set((job.contextualStrings ?? []).map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty }))
    if !contextualStrings.isEmpty {
        analysisContext.contextualStrings[.general] = contextualStrings
    }
    let audioFile = try AVAudioFile(forReading: inputURL)
    try await analyzer.setContext(analysisContext)
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
