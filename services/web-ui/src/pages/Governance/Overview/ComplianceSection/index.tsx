import {
    Button,
    Card,
    Flex,
    Grid,
    ProgressCircle,
    Subtitle,
    Text,
    Title,
} from '@tremor/react'
import { useNavigate, useParams } from 'react-router-dom'
import { ChevronRightIcon } from '@heroicons/react/20/solid'
import { useAtomValue } from 'jotai'
import { useComplianceApiV1BenchmarksSummaryList } from '../../../../api/compliance.gen'
import {
    errorHandlingWithErrorMessage,
    getErrorMessage,
    toErrorMessage,
} from '../../../../types/apierror'
import { searchAtom } from '../../../../utilities/urlstate'
import { getConnectorsIcon } from '../../../../components/Cards/ConnectorCard'

function CorrespondingColor(value: number) {
    if (value < 25) return 'rose'
    if (value >= 25 && value < 50) return 'amber'
    if (value >= 50) return 'emerald'
    return 'slate'
}

function ValueCalculator(bs: any) {
    return (
        ((bs?.controlsSeverityStatus?.total?.passed || 0) /
            (bs?.controlsSeverityStatus?.total?.total || 1)) *
        100
    )
}

export default function Compliance() {
    const workspace = useParams<{ ws: string }>().ws
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)

    const {
        response: benchmarks,
        isLoading,
        error,
        sendNow: refresh,
    } = useComplianceApiV1BenchmarksSummaryList()

    const sorted = [...(benchmarks?.benchmarkSummary || [])].filter((b) => {
        const ent = Object.entries(b.tags || {})
        return (
            ent.filter(
                (v) =>
                    v[0] === 'kaytu_benchmark_type' && v[1][0] === 'compliance'
            ).length > 0
        )
    })
    sorted.sort((a, b) => {
        const bScore =
            (b?.controlsSeverityStatus?.total?.passed || 0) /
            (b?.controlsSeverityStatus?.total?.total || 1)
        const aScore =
            (a?.controlsSeverityStatus?.total?.passed || 0) /
            (a?.controlsSeverityStatus?.total?.total || 1)

        const aZero = (a?.controlsSeverityStatus?.total?.total || 0) === 0
        const bZero = (b?.controlsSeverityStatus?.total?.total || 0) === 0

        if ((aZero && bZero) || aScore === bScore) {
            return 0
        }
        if (aZero) {
            return 1
        }
        if (bZero) {
            return -1
        }
        return aScore > bScore ? 1 : -1
    })

    return (
        <Card>
            {' '}
            <Flex flexDirection="col" alignItems="start" justifyContent="start">
                <Flex className="mb-6">
                    <text className="text-lg font-bold">Benchmarks</text>
                    <Button
                        variant="light"
                        icon={ChevronRightIcon}
                        iconPosition="right"
                        onClick={() =>
                            navigate(
                                `/compliance?${searchParams}`
                            )
                        }
                    >
                        Show all
                    </Button>
                </Flex>
                {isLoading || getErrorMessage(error).length > 0 ? (
                    <Grid numItems={2} className="gap-4 w-full">
                        {[1, 2, 3, 4].map((i) => {
                            return (
                                <Card className="p-3 dark:ring-gray-500">
                                    <Flex
                                        flexDirection="col"
                                        alignItems="start"
                                        justifyContent="start"
                                        className="animate-pulse w-full"
                                    >
                                        <ProgressCircle value={0} size="md">
                                            <div className="animate-pulse h-8 w-8 my-2 bg-slate-200 dark:bg-slate-700 rounded-full" />
                                        </ProgressCircle>
                                        <div className="h-5 w-full mt-3 mb-2 bg-slate-200 dark:bg-slate-700 rounded" />
                                        <div className="h-6 w-full bg-slate-200 dark:bg-slate-700 rounded" />
                                    </Flex>
                                </Card>
                            )
                        })}
                    </Grid>
                ) : (
                    <Grid numItems={2} className="gap-4">
                        {sorted?.map(
                            (bs, i) =>
                                i < 4 && (
                                    <Card
                                        onClick={() =>
                                            navigate(
                                                `/compliance/${bs.id}?${searchParams}`
                                            )
                                        }
                                        className="p-3 cursor-pointer dark:ring-gray-500 hover:shadow-md"
                                    >
                                        <Flex className="w-full">
                                            <ProgressCircle
                                                value={ValueCalculator(bs)}
                                                size="md"
                                                color={CorrespondingColor(
                                                    ValueCalculator(bs)
                                                )}
                                            >
                                                {bs?.tags?.kaytu_logo
                                                    ? bs?.tags?.kaytu_logo.map(
                                                          (logo) => (
                                                              <div className="min-w-[36px] w-9 h-9 rounded-full overflow-hidden border border-gray-100">
                                                                  <img
                                                                      className="w-full"
                                                                      alt={logo}
                                                                      src={logo}
                                                                  />
                                                              </div>
                                                          )
                                                      )
                                                    : getConnectorsIcon(
                                                          bs?.connectors || []
                                                      )}
                                            </ProgressCircle>
                                        </Flex>

                                        <Text className=" text-gray-800 mt-3 mb-1 truncate">
                                            {bs.title}
                                        </Text>
                                        {(bs.controlsSeverityStatus?.total
                                            ?.total || 0) > 0 && (
                                            <Title
                                                className={`font-bold text-${CorrespondingColor(
                                                    ValueCalculator(bs)
                                                )}-500`}
                                            >
                                                {(
                                                    ValueCalculator(bs) || 0
                                                ).toFixed(0)}
                                                <span className="font-medium text-xs ml-0.5">
                                                    %
                                                </span>
                                            </Title>
                                        )}
                                    </Card>
                                )
                        )}

                        {errorHandlingWithErrorMessage(
                            refresh,
                            toErrorMessage(error)
                        )}
                    </Grid>
                )}
            </Flex>
        </Card>
    )
}
