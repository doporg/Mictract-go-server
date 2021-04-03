Organizations:
    - &Org{{.ID}}
        Name: org{{.ID}}MSP
        ID: org{{.ID}}MSP
        MSPDir: /mictract/networks/net{{.NetworkID}}/peerOrganizations/org{{.ID}}.net{{.NetworkID}}.com/msp
        Policies:
            Readers:
                Type: Signature
                Rule: "OR('org{{.ID}}MSP.admin', 'org{{.ID}}MSP.peer', 'org{{.ID}}MSP.client')"
            Writers:
                Type: Signature
                Rule: "OR('org{{.ID}}MSP.admin', 'org{{.ID}}MSP.client')"
            Admins:
                Type: Signature
                Rule: "OR('org{{.ID}}MSP.admin')"
            Endorsement:
                Type: Signature
                Rule: "OR('org{{.ID}}MSP.peer')"

        AnchorPeers:
            - Host: peer1-org{{.ID}}-net{{.NetworkID}}
              Port: 7051

Capabilities:
    Channel: &ChannelCapabilities
        V2_0: true
    Orderer: &OrdererCapabilities
        V2_0: true
    Application: &ApplicationCapabilities
        V2_0: true

Application: &ApplicationDefaults
    Organizations:
    Policies:
        Readers:
            Type: ImplicitMeta
            Rule: "ANY Readers"
        Writers:
            Type: ImplicitMeta
            Rule: "ANY Writers"
        Admins:
            Type: ImplicitMeta
            Rule: "MAJORITY Admins"
        LifecycleEndorsement:
            Type: ImplicitMeta
            Rule: "MAJORITY Endorsement"
        Endorsement:
            Type: ImplicitMeta
            Rule: "MAJORITY Endorsement"

    Capabilities:
        <<: *ApplicationCapabilities

Channel: &ChannelDefaults
    Policies:
        Readers:
            Type: ImplicitMeta
            Rule: "ANY Readers"
        Writers:
            Type: ImplicitMeta
            Rule: "ANY Writers"
        Admins:
            Type: ImplicitMeta
            Rule: "MAJORITY Admins"
    Capabilities:
        <<: *ChannelCapabilities

Profiles:
    NewChannel:
        Consortium: LLJConsortium
        <<: *ChannelDefaults
        Application:
            <<: *ApplicationDefaults
            Organizations:
                - *Org{{.ID}}
            Capabilities:
                <<: *ApplicationCapabilities
