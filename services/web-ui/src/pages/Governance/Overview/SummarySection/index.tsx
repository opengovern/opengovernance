import { Badge, Card, Flex, Text, Title } from '@tremor/react'
import { ArrowDownFill } from '../../../../icons/icons'
import LightBadge from '../../../../components/LightBadge'
import { useComplianceApiV1ControlsSummaryList } from '../../../../api/compliance.gen'

export default function SummarySection() {
    const { response, isLoading, isExecuted, error } =
        useComplianceApiV1ControlsSummaryList()

    const total = new Map<string, number>()
    const passed = new Map<string, number>()
    response?.forEach((i) => {
        Object.entries(i.control?.tags || {})
            .filter((v) => v[0] === 'score_tags')
            .flatMap((v) => v[1])
            .forEach((v) => {
                if (i.passed) {
                    passed.set(v, (passed.get(v) || 0) + 1)
                }
                total.set(v, (total.get(v) || 0) + 1)
            })
    })

    const SummaryList = [
        {
            title: 'Cloud Accounts with MFA ',
            value: 45,
            change: 10,
        },
        {
            title: 'Subnets with Unrestricted Traffic',
            value: 23,
            change: -10,
        },
        { title: 'Unrecoverable KeyVaults', value: 31, change: 0 },
        { title: 'Problematic IAM Roles', value: 19, change: -25 },
        { title: 'Non-SSO Users', value: 28, change: 15 },
        { title: 'Cloud Access without MFA', value: 116, change: 80 },
        { title: 'Missing Audit Logs', value: 6, change: -5 },
    ] /* .map((i) => {
        return {
            title: i.title,
            value: passed.get(i.title)||0,
            score: (passed.get(i.title) || 0) / (total.get(i.title) || 1),
        }
    }) */

    return (
        <Card>
            <text className="text-lg font-bold">Summary</text>
            <Flex flexDirection="col" alignItems="start" className=" mt-4">
                {SummaryList.map((i) => (
                    <Flex
                        className={`py-1.5 ${
                            i.title ===
                            SummaryList[SummaryList.length - 1].title
                                ? 'pb-0'
                                : 'border-b border-gray-100'
                        } `}
                    >
                        <Text className="w-full">{i.title}</Text>
                        <Flex className="w-fit gap-4">
                            <Text className="font-bold text-gray-800">
                                {i.value}
                            </Text>
                            <Flex className="w-12">
                                <LightBadge value={i.change} />
                            </Flex>
                        </Flex>
                    </Flex>
                ))}
            </Flex>
        </Card>
    )
}
