import SwiftUI

struct ConnectView: View {
    @StateObject private var viewModel = ConnectionViewModel()
    @AppStorage("signalingURL") private var signalingURL = "ws://192.168.1.100:8080"

    var body: some View {
        NavigationStack {
            switch viewModel.state {
            case .disconnected, .connecting:
                connectForm
            case .registered, .selectingHost:
                hostList
            case .connected:
                RemoteDisplayView(viewModel: viewModel)
            }
        }
        .alert("Error", isPresented: .constant(viewModel.errorMessage != nil)) {
            Button("OK") { viewModel.errorMessage = nil }
        } message: {
            Text(viewModel.errorMessage ?? "")
        }
    }

    // MARK: - Connect form

    private var connectForm: some View {
        VStack(spacing: 24) {
            Spacer()

            Text("AirMac")
                .font(.largeTitle.bold())

            Text("Control your Mac from iPhone")
                .font(.subheadline)
                .foregroundStyle(.secondary)

            VStack(alignment: .leading, spacing: 8) {
                Text("Signaling Server URL")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                TextField("ws://192.168.1.100:8080", text: $signalingURL)
                    .textFieldStyle(.roundedBorder)
                    .autocorrectionDisabled()
                    .textInputAutocapitalization(.never)
                    .keyboardType(.URL)
            }
            .padding(.horizontal)

            Button {
                viewModel.connect(signalingURL: signalingURL)
            } label: {
                HStack {
                    if viewModel.state == .connecting {
                        ProgressView()
                            .tint(.white)
                    }
                    Text(viewModel.state == .connecting ? "Connecting..." : "Connect")
                }
                .frame(maxWidth: .infinity)
                .padding()
                .background(Color.accentColor)
                .foregroundColor(.white)
                .cornerRadius(12)
            }
            .disabled(viewModel.state == .connecting || signalingURL.isEmpty)
            .padding(.horizontal)

            Spacer()
        }
        .navigationTitle("")
    }

    // MARK: - Host selection list

    private var hostList: some View {
        List {
            if viewModel.hosts.isEmpty {
                HStack {
                    ProgressView()
                    Text("Waiting for hosts...")
                        .foregroundStyle(.secondary)
                }
            } else {
                ForEach(viewModel.hosts) { host in
                    Button {
                        viewModel.selectHost(host.id)
                    } label: {
                        HStack {
                            Image(systemName: "desktopcomputer")
                                .foregroundStyle(host.online ? .green : .red)
                            Text(host.id)
                            Spacer()
                            if host.online {
                                Image(systemName: "chevron.right")
                                    .foregroundStyle(.secondary)
                            }
                        }
                    }
                    .disabled(!host.online)
                }
            }
        }
        .navigationTitle("Select Host")
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button("Disconnect") {
                    viewModel.disconnect()
                }
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    viewModel.refreshHostList()
                } label: {
                    Image(systemName: "arrow.clockwise")
                }
            }
        }
    }
}
