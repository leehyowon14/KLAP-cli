// swift-tools-version: 6.0

import PackageDescription

let package = Package(
    name: "KLAPMacOSBridges",
    platforms: [
        .macOS(.v15)
    ],
    products: [
        .executable(name: "TranscriptBridge", targets: ["TranscriptBridge"])
    ],
    targets: [
        .executableTarget(name: "TranscriptBridge")
    ]
)
