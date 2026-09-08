import Foundation
import Testing
@testable import EventKitCore

struct BridgeIOTests {
    struct Request: Codable {
        let dueAt: Date?
        let title: String
    }

    @Test func dateAndOptionalJSONContract() throws {
        let request = try decodeBridgeRequest(Request.self, from: Data(#"{"dueAt":"2026-03-02T00:00:00Z","title":"개강"}"#.utf8))
        #expect(request.dueAt != nil)
        #expect(request.title == "개강")
        let empty = try decodeBridgeRequest(Request.self, from: Data(#"{"dueAt":null,"title":""}"#.utf8))
        #expect(empty.dueAt == nil)
        #expect(String(data: try encodeBridgeResponse(empty), encoding: .utf8) == #"{"title":""}"#)
        #expect(throws: (any Error).self) {
            try decodeBridgeRequest(Request.self, from: Data("broken".utf8))
        }
    }

    @Test func permissionsPreserveInputOrderWithoutEventKit() throws {
        let result = try collectPermissions(count: 3) { index, callback in
            DispatchQueue.global().async { callback(index != 1, nil) }
        }
        #expect(result == [true, false, true])
        let empty = try collectPermissions(count: 0) { _, _ in Issue.record("unexpected request") }
        #expect(empty.isEmpty)
    }

    @Test func permissionFailureIsNotDenial() {
        enum Failure: Error { case fixture }
        #expect(throws: Failure.self) {
            try collectPermissions(count: 2) { index, callback in
                callback(false, index == 0 ? Failure.fixture : nil)
            }
        }
    }
}
