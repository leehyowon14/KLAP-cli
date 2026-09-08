// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "KLAPMacOSBridges",
    platforms: [
        .macOS("26")
    ],
    products: [
        .executable(name: "TranscriptBridge", targets: ["TranscriptBridge"]),
        .executable(name: "ReminderBridge", targets: ["ReminderBridge"])
    ],
    targets: [
        .executableTarget(name: "TranscriptBridge"),
        // Preserve the distributed EventKit scripts' Swift 5 language mode.
        .target(name: "EventKitCore", swiftSettings: [.swiftLanguageMode(.v5)]),
        .executableTarget(name: "ReminderBridge", dependencies: ["EventKitCore"], swiftSettings: [.swiftLanguageMode(.v5)]),
        .testTarget(name: "EventKitCoreTests", dependencies: ["EventKitCore"], swiftSettings: [.swiftLanguageMode(.v5)])
    ]
)
