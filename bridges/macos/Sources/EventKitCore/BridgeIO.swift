import EventKit
import Foundation

public func decodeBridgeRequest<T: Decodable>(_ type: T.Type, from data: Data) throws -> T {
    let decoder = JSONDecoder()
    decoder.dateDecodingStrategy = .iso8601
    return try decoder.decode(type, from: data)
}

public func encodeBridgeResponse<T: Encodable>(_ value: T) throws -> Data {
    let encoder = JSONEncoder()
    encoder.outputFormatting = [.sortedKeys]
    return try encoder.encode(value)
}

// Start all permission requests before waiting, preserving Category's parallel
// permission requests. Results keep input order and the first reported error.
func collectPermissions(count: Int, request: (Int, @escaping (Bool, Error?) -> Void) -> Void) throws -> [Bool] {
    let group = DispatchGroup()
    let lock = NSLock()
    var granted = Array(repeating: false, count: count)
    var firstError: Error?
    for index in 0..<count {
        group.enter()
        request(index) { ok, error in
            lock.lock()
            granted[index] = ok
            if firstError == nil { firstError = error }
            lock.unlock()
            group.leave()
        }
    }
    group.wait()
    if let firstError { throw firstError }
    return granted
}

public func requestPermissions(_ types: [EKEntityType], from store: EKEventStore) throws -> [Bool] {
    try collectPermissions(count: types.count) { index, completion in
        if #available(macOS 14.0, *) {
            switch types[index] {
            case .reminder: store.requestFullAccessToReminders(completion: completion)
            case .event: store.requestFullAccessToEvents(completion: completion)
            @unknown default: completion(false, NSError(domain: "KLAPEventKitCore", code: 1, userInfo: [NSLocalizedDescriptionKey: "Unsupported EventKit entity type"]))
            }
        } else {
            store.requestAccess(to: types[index], completion: completion)
        }
    }
}
