import AVFoundation
import CoreMedia
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
    let progress: Bool?
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

struct TranscribeEvent: Codable {
    let type: String
    let inputPath: String?
    let outputPath: String?
    let progress: Double?
    let error: String?
    let results: [TranscribeResult]?
}

func emitEvent(_ event: TranscribeEvent) {
    guard let data = try? encodeJSON(event) else {
        return
    }
    FileHandle.standardOutput.write(data)
    FileHandle.standardOutput.write(Data("\n".utf8))
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
func locale(for job: TranscribeJob) -> Locale {
    let localeIdentifier = job.locale?.trimmingCharacters(in: .whitespacesAndNewlines)
    return Locale(identifier: localeIdentifier?.isEmpty == false ? localeIdentifier! : "ko-KR")
}

@available(macOS 26.0, *)
func prepareSpeechAssets(for locale: Locale) async throws {
    guard SpeechTranscriber.isAvailable else {
        throw BridgeError.transcriberUnavailable
    }

    let requestedLocale = locale.identifier(.bcp47)
    let supportedLocales = await SpeechTranscriber.supportedLocales
    guard supportedLocales.contains(where: { $0.identifier(.bcp47) == requestedLocale }) else {
        throw BridgeError.unsupportedLocale(requestedLocale)
    }

    let transcriber = makeSpeechTranscriber(locale: locale)
    let modules: [any SpeechModule] = [transcriber]

    let installedLocales = await SpeechTranscriber.installedLocales
    if !installedLocales.contains(where: { $0.identifier(.bcp47) == requestedLocale }) {
        if let request = try await AssetInventory.assetInstallationRequest(supporting: modules) {
            try await request.downloadAndInstall()
        }
    }
}

@available(macOS 26.0, *)
func makeSpeechTranscriber(locale: Locale) -> SpeechTranscriber {
    SpeechTranscriber(
        locale: locale,
        transcriptionOptions: [],
        reportingOptions: [.volatileResults],
        attributeOptions: []
    )
}

@available(macOS 26.0, *)
func releaseReservedSpeechAssets() async {
    for reservedLocale in await AssetInventory.reservedLocales {
        await AssetInventory.release(reservedLocale: reservedLocale)
    }
}

@available(macOS 26.0, *)
func transcribe(job: TranscribeJob, emitProgress: Bool) async throws -> TranscribeResult {
    let inputURL = URL(fileURLWithPath: job.inputPath)
    let outputURL = URL(fileURLWithPath: job.outputPath)
    let locale = locale(for: job)

    guard FileManager.default.fileExists(atPath: inputURL.path) else {
        throw CocoaError(.fileNoSuchFile)
    }

    let transcriber = makeSpeechTranscriber(locale: locale)
    let modules: [any SpeechModule] = [transcriber]

    let analyzer = SpeechAnalyzer(modules: modules)
    let analysisContext = AnalysisContext()
    let contextualStrings = Array(Set((job.contextualStrings ?? []).map { $0.trimmingCharacters(in: .whitespacesAndNewlines) }.filter { !$0.isEmpty }))
    if !contextualStrings.isEmpty {
        analysisContext.contextualStrings[.general] = contextualStrings
    }
    let audioFile = try AVAudioFile(forReading: inputURL)
    let duration = audioFile.fileFormat.sampleRate > 0 ? Double(audioFile.length) / audioFile.fileFormat.sampleRate : 0
    try await analyzer.setContext(analysisContext)
    try await analyzer.start(inputAudioFile: audioFile, finishAfterFile: true)

    var transcript = AttributedString("")
    for try await result in transcriber.results {
        if emitProgress {
            let endSeconds = CMTimeGetSeconds(result.resultsFinalizationTime)
            if endSeconds.isFinite && duration > 0 {
                emitEvent(TranscribeEvent(
                    type: "progress",
                    inputPath: job.inputPath,
                    outputPath: job.outputPath,
                    progress: min(max(endSeconds / duration, 0), 0.99),
                    error: nil,
                    results: nil
                ))
            }
        }
        if result.isFinal {
            transcript += result.text
        }
    }

    let text = String(transcript.characters)
    try FileManager.default.createDirectory(at: outputURL.deletingLastPathComponent(), withIntermediateDirectories: true)
    try text.write(to: outputURL, atomically: true, encoding: .utf8)
    return TranscribeResult(inputPath: job.inputPath, outputPath: job.outputPath, text: text, error: nil)
}


func encodeJSON<T: Encodable>(_ value: T) throws -> Data {
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.sortedKeys]
    return try encoder.encode(value)
}

func encodeTranscriptResponse(_ results: [TranscribeResult], progress: Bool) throws -> Data {
    if progress {
        var data = try encodeJSON(TranscribeEvent(type: "response", inputPath: nil, outputPath: nil, progress: nil, error: nil, results: results))
        data.append(Data("\n".utf8))
        return data
    }
    return try encodeJSON(TranscribeResponse(results: results))
}

public func runTranscriptBridge(input: Data) async throws -> Data {
    let request = try JSONDecoder().decode(TranscribeRequest.self, from: input)
    let shouldEmitProgress = request.progress ?? false
    // Empty protocol probes must not touch Speech assets or device permissions.
    if request.jobs.isEmpty {
        return try encodeTranscriptResponse([], progress: shouldEmitProgress)
    }
    var results: [TranscribeResult] = []
    var preparedLocales: Set<String> = []
    if #available(macOS 26.0, *) {
        await releaseReservedSpeechAssets()
    }
    for job in request.jobs {
        do {
            if #available(macOS 26.0, *) {
                let locale = locale(for: job)
                let localeKey = locale.identifier(.bcp47)
                if !preparedLocales.contains(localeKey) {
                    try await prepareSpeechAssets(for: locale)
                    preparedLocales.insert(localeKey)
                }
                results.append(try await transcribe(job: job, emitProgress: shouldEmitProgress))
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


    return try encodeTranscriptResponse(results, progress: shouldEmitProgress)
}
