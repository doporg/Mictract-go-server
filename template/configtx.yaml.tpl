Organizations:
    - &OrdererOrg
        Name: ordererorg
        ID: ordererMSP
        MSPDir: /mictract/networks/net{{.NetworkID}}/ordererOrganizations/net{{.NetworkID}}.com/msp
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
            - orderer{{.ID}}-net{{.NetworkID}}:7050

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
    OrdererType: etcdraft
    Addresses:
        - orderer{{.ID}}-net{{.NetworkID}}:7050
    EtcdRaft:
        Consenters:
        - Host: orderer1-net{{.ID}}
          Port: 7050
          ClientTLSCert: /mictract/networks/net{{.NetworkID}}/ordererOrganizations/net{{.NetworkID}}.com/orderers/orderer{{.ID}}.net{{.NetworkID}}.com/tls/server.crt
          ServerTLSCert: /mictract/networks/net{{.NetworkID}}/ordererOrganizations/net{{.NetworkID}}.com/orderers/orderer{{.ID}}.net{{.NetworkID}}.com/tls/server.crt
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
            LLJConsortium:
                Organizations:
