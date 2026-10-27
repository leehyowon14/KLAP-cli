import Foundation
import Testing
@testable import SpeechCore

struct ReservationTests {
    @Test func releasesOnlyNewReservationsOnSuccessFailureAndCancellation() async throws {
        guard #available(macOS 26.0, *) else { return }
        for acquired in [true, false] {
            for fails in [true, false] {
                var events: [String] = []
                do {
                    let value = try await withSpeechAssetReservation(locale: Locale(identifier: "ko-KR"), reserve: { _ in
                        events.append("reserve"); return acquired
                    }, release: { _ in
                        events.append("release")
                    }, operation: {
                        events.append("work")
                        if fails { throw CancellationError() }
                        return 42
                    })
                    #expect(!fails && value == 42)
                } catch is CancellationError { #expect(fails) }
                #expect(events == (acquired ? ["reserve", "work", "release"] : ["reserve", "work"]))
            }
        }
    }
    @Test func failedReservationDoesNotStartWorkOrRelease() async {
        guard #available(macOS 26.0, *) else { return }
        var worked = false
        var released = false
        do {
            let _: Int = try await withSpeechAssetReservation(locale: Locale(identifier: "ko-KR"), reserve: { _ in
                throw CancellationError()
            }, release: { _ in released = true }, operation: { worked = true; return 1 })
            Issue.record("Failed reservation unexpectedly succeeded")
        } catch {}
        #expect(!worked && !released)
    }
}
