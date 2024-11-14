import { Badge, Card, Col, Flex, Grid, Text, Title } from '@tremor/react'
import TopHeader from '../../../components/Layout/Header'
import ComplianceSection from './ComplianceSection'
import SummarySection from './SummarySection'
import AdvancedKPIGroup from '../../../components/KPIGroup/AdvancedKPIGroup'
import SimpleKPIGroup from '../../../components/KPIGroup/SimpleKPIGroup'
import { IKPISingleItem } from '../../../components/KPIGroup/KPISingleItem'

export default function SecurityOverview() {
    const CloudAccessKPIs: IKPISingleItem[] = [
        { title: 'Users with Non-Compliant MFA', value: 45, change: 13 },
        { title: 'Non-Compliant API Keys', value: 87, change: 50 },
        { title: 'Duplicate Access', value: 37, change: -25 },
        { title: 'Users with Excessive Permissions', value: 94, change: -5 },
    ]
    const NetworkKPIs: IKPISingleItem[] = [
        { title: 'Load Balancers with no WAF', value: 89, change: 33 },
        { title: 'VMs on Public Subnet', value: 34, change: 0 },
        { title: 'Insecure Certificate', value: 61, change: 5 },
        {
            title: 'VM with FTP/SMTP/RDP/SSH Open',
            value: 19,
            change: -10,
        },
    ]

    const DataKPIs: IKPISingleItem[] = [
        { title: 'Unencrypted Storage', value: 29, change: 10 },
        { title: 'Systems with no backup', value: 63, change: -15 },
        { title: 'Internet Accessible Databases', value: 26, change: 0 },
        {
            title: 'Storage with no Authentication',
            value: 22,
            change: 25,
        },
    ]

    const WebAndAppKPIs: IKPISingleItem[] = [
        { title: 'Users with Non-Compliant MFA', value: 12, change: 12 },
        { title: 'Non-Compliant API Keys', value: 12, change: 12 },
    ]

    return (
        <>
            <TopHeader
                supportedFilters={['Cloud Account']}
                initialFilters={['Cloud Account']}
            />
            <Flex>
                <Grid numItems={5} className="w-full gap-4">
                    <Col numColSpan={2}>
                        <Flex
                            alignItems="start"
                            flexDirection="col"
                            className="gap-4"
                        >
                            <SummarySection />
                            <ComplianceSection />
                        </Flex>
                    </Col>
                    <Col numColSpan={3}>
                        <Flex
                            alignItems="start"
                            flexDirection="col"
                            className="gap-4"
                        >
                            <AdvancedKPIGroup
                                mainTitle="Cloud Access"
                                mainValue={68}
                                mainChange={12}
                                otherKpis={CloudAccessKPIs}
                            />
                            <AdvancedKPIGroup
                                mainTitle="Network"
                                mainValue={97}
                                mainChange={-20}
                                otherKpis={NetworkKPIs}
                            />
                            <AdvancedKPIGroup
                                mainTitle="Data"
                                mainValue={113}
                                mainChange={0}
                                otherKpis={DataKPIs}
                            />
                            <Flex className="gap-4">
                                <SimpleKPIGroup
                                    mainTitle="Web & Application"
                                    otherKpis={WebAndAppKPIs}
                                />
                                <SimpleKPIGroup
                                    mainTitle="Endpoints"
                                    otherKpis={WebAndAppKPIs}
                                />
                            </Flex>
                        </Flex>
                    </Col>
                </Grid>
            </Flex>
        </>
    )
}
