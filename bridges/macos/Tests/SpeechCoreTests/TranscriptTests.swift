import Foundation
import Testing
@testable import SpeechCore

struct TranscriptTests {
    func fixture(_ name: String) throws -> Data {
        let root = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
        return try Data(contentsOf: root.appendingPathComponent("testdata").appendingPathComponent(name))
    }

    @Test func emptyJobsDoNotUseSpeechAndMatchFixtures() async throws {
        for pair in [
            ("empty-request.json", "empty-response.json"),
            ("empty-progress-request.json", "empty-progress-response.ndjson"),
        ] {
            let output = try await runTranscriptBridge(input: fixture(pair.0))
            let expected = try fixture(pair.1)
            #expect(String(decoding: output, as: UTF8.self).trimmingCharacters(in: .whitespacesAndNewlines)
                == String(decoding: expected, as: UTF8.self).trimmingCharacters(in: .whitespacesAndNewlines))
        }
    }

    @Test func malformedRequestFailsBeforeSpeech() async {
        await #expect(throws: (any Error).self) {
            try await runTranscriptBridge(input: Data("broken".utf8))
        }
    }

    @Test func populatedResponseMatchesGoFixture() throws {
        let expected = try JSONDecoder().decode(TranscribeResponse.self, from: fixture("response.json"))
        let encoded = try encodeTranscriptResponse(expected.results, progress: false)
        let roundTrip = try JSONDecoder().decode(TranscribeResponse.self, from: encoded)
        #expect(roundTrip.results.count == 1)
        #expect(roundTrip.results[0].text == "전사 결과")
        let event = try JSONDecoder().decode(TranscribeEvent.self, from: encodeTranscriptResponse(expected.results, progress: true))
        #expect(event.type == "response")
        #expect(event.results?.first?.text == "전사 결과")
    }
}
