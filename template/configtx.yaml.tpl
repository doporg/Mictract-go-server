Organizations:
    - &OrdererOrg
        Name: ordererorg
        ID: ordererMSP
        MSPDir: /mictract/networks/net{{.ID}}/ordererOrganizations/net{{.ID}}.com/msp
        Policies:
            Readers:
                Type: Signature
                Rule: "OR('ordererMSP.member')"
            Writers:
                Type: Signature
                Rule: "OR('ordererMSP.member')"
            Admins:
                Type: Signature
                Rule: "OR('ordererMSP.admin')"

        OrdererEndpoints:
            - orderer1-net{{.ID}}:7050

    - &Org1
        Name: org1
        ID: org1MSP
        MSPDir: /mictract/networks/net{{.ID}}/peerOrganizations/org1.net{{.ID}}.com/msp
        Policies:
            Readers:
                Type: Signature
                Rule: "OR('org1MSP.admin', 'org1MSP.peer', 'org1MSP.client')"
            Writers:
                Type: Signature
                Rule: "OR('org1MSP.admin', 'org1MSP.client')"
            Admins:
                Type: Signature
                Rule: "OR('org1MSP.admin')"
            Endorsement:
                Type: Signature
                Rule: "OR('org1MSP.peer')"

        AnchorPeers:
            - Host: peer1-org1-net{{.ID}}
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

Orderer: &OrdererDefaults
    OrdererType: {{.Consensus}}
    Addresses:
        - orderer1-net{{.ID}}:7050
    EtcdRaft:
        Consenters:
        - Host: orderer1-net{{.ID}}
          Port: 7050
          ClientTLSCert: /mictract/networks/net{{.ID}}/ordererOrganizations/net{{.ID}}.com/orderers/orderer1.net{{.ID}}.com/tls/server.crt
          ServerTLSCert: /mictract/networks/net{{.ID}}/ordererOrganizations/net{{.ID}}.com/orderers/orderer1.net{{.ID}}.com/tls/server.crt
    BatchTimeout: 2s
    BatchSize:
        MaxMessageCount: 10
        AbsoluteMaxBytes: 99 MB
        PreferredMaxBytes: 512 KB
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
        BlockValidation:
            Type: ImplicitMeta
            Rule: "ANY Writers"

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
    Genesis:
        <<: *ChannelDefaults
        Orderer:
            <<: *OrdererDefaults
            Organizations:
                - *OrdererOrg
            Capabilities:
                <<: *OrdererCapabilities
        Consortiums:
            SampleConsortium:
                Organizations:
                    - *Org1
    DefaultChannel:
        Consortium: SampleConsortium
        <<: *ChannelDefaults
        Application:
            <<: *ApplicationDefaults
            Organizations:
                - *Org1
            Capabilities:
                <<: *ApplicationCapabilities
