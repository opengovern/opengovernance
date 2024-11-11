import { Link, useNavigate, useParams } from 'react-router-dom'
import { useSetAtom } from 'jotai'
import { useEffect, useState } from 'react'
import {
    Button,
    Card,
    Flex,
    Grid,
    Tab,
    TabGroup,
    TabList,
    Text,
    Title,
    Badge,
    Metric,
    Icon,
    TabPanels,
    TabPanel,
} from '@tremor/react'
import MarkdownPreview from '@uiw/react-markdown-preview'
import Editor from 'react-simple-code-editor'
import { highlight, languages } from 'prismjs'
import {
    DocumentDuplicateIcon,
    DocumentTextIcon,
    CommandLineIcon,
    Square2StackIcon,
    ClockIcon,
    CodeBracketIcon,
    Cog8ToothIcon,
    BookOpenIcon,
    PencilIcon,
} from '@heroicons/react/24/outline'
import clipboardCopy from 'clipboard-copy'
import { ChevronRightIcon } from '@heroicons/react/20/solid'
import { notificationAtom, queryAtom } from '../../../store'
import { dateTimeDisplay } from '../../../utilities/dateDisplay'
import {
    useComplianceApiV1BenchmarksSummaryDetail,
    useComplianceApiV1ControlsSummaryDetail,
} from '../../../api/compliance.gen'
import Spinner from '../../../components/Spinner'
import Modal from '../../../components/Modal'
import TopHeader from '../../../components/Layout/Header'
import { useFilterState } from '../../../utilities/urlstate'
import SummaryCard from '../../../components/Cards/SummaryCard'
import EaseOfSolutionChart from '../../../components/EaseOfSolutionChart'
import ImpactedResources from '../../Governance/Controls/ControlSummary/Tabs/ImpactedResources'
import ImpactedAccounts from '../../Governance/Controls/ControlSummary/Tabs/ImpactedAccounts'
import { severityBadge } from '../../Governance/Controls'
import {
    GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus,
    SourceType,
} from '../../../api/api'
import DrawerPanel from '../../../components/DrawerPanel'
import { numberDisplay } from '../../../utilities/numericDisplay'
import { useMetadataApiV1QueryParameterList } from '../../../api/metadata.gen'
import {
    useScheduleApiV1ComplianceReEvaluateUpdate,
    useScheduleApiV1ComplianceTriggerUpdate,
} from '../../../api/schedule.gen'

export default function ScoreDetails() {
    const { id, ws } = useParams()
    const { value: selectedConnections } = useFilterState()
    const navigate = useNavigate()

    const [doc, setDoc] = useState('')
    const [docTitle, setDocTitle] = useState('')
    const [modalData, setModalData] = useState('')
    const setNotification = useSetAtom(notificationAtom)
    const setQuery = useSetAtom(queryAtom)
    const [hideTabs, setHideTabs] = useState(false)
    const [selectedTabIndex, setSelectedTabIndex] = useState(0)

    const { response: controlDetail, isLoading } =
        useComplianceApiV1ControlsSummaryDetail(String(id), {
            connectionId: selectedConnections.connections,
        })

    const {
        response: parameters,
        isLoading: parametersLoading,
        isExecuted,
        sendNow: refresh,
    } = useMetadataApiV1QueryParameterList()

    const benchmarkID = controlDetail?.benchmarks?.at(0)?.id || ''

    const {
        error: reevaluateError,
        isLoading: isReevaluateLoading,
        isExecuted: isReevaluateExecuted,
        sendNow: ReEvaluate,
    } = useScheduleApiV1ComplianceTriggerUpdate(
        {
            benchmark_id: [benchmarkID],
            connection_id: [],
        },
        {},
        false
    )

    const {
        response: benchmarkDetail,
        isLoading: benchmarkDetailsLoading,
        isExecuted: benchmarkDetailsExecuted,
        sendNow: refreshBenchmark,
    } = useComplianceApiV1BenchmarksSummaryDetail(
        benchmarkID,
        {},
        {},
        benchmarkID !== ''
    )

    useEffect(() => {
        if (isReevaluateExecuted && !isReevaluateLoading) {
            refreshBenchmark()
        }
    }, [isReevaluateLoading])

    const isJobRunning =
        benchmarkDetail?.lastJobStatus !== 'FAILED' &&
        benchmarkDetail?.lastJobStatus !== 'SUCCEEDED' &&
        (benchmarkDetail?.lastJobStatus || '') !== ''

    const reEvaluateButtonLoading =
        (isReevaluateExecuted && isReevaluateLoading) ||
        (benchmarkDetailsExecuted && benchmarkDetailsLoading) ||
        (benchmarkDetailsExecuted && isJobRunning)

    const costSaving = controlDetail?.costOptimization || 0

    const customizableQuery =
        (controlDetail?.control?.query?.parameters?.length || 0) > 0 ||
        (controlDetail?.control?.query?.queryToExecute || '').includes('{{')

    const [conformanceFilter, setConformanceFilter] = useState<
        | GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus[]
        | undefined
    >(undefined)
    const conformanceFilterIdx = () => {
        if (
            conformanceFilter?.length === 1 &&
            conformanceFilter[0] ===
                GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus.ConformanceStatusFailed
        ) {
            return 1
        }
        if (
            conformanceFilter?.length === 1 &&
            conformanceFilter[0] ===
                GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus.ConformanceStatusFailed
        ) {
            return 2
        }
        return 0
    }

    return (
        <>
            <TopHeader
                breadCrumb={[
                    !isLoading ? controlDetail?.control?.title : 'Score detail',
                ]}
                supportedFilters={['Cloud Account']}
            />
            {isLoading || parametersLoading ? (
                <Flex justifyContent="center" className="mt-56">
                    <Spinner />
                </Flex>
            ) : (
                <>
                    <Flex flexDirection="col" className="mb-8 mt-4 gap-4">
                        <Flex justifyContent="start" className="gap-4 w-full">
                            <Metric className="font-semibold whitespace-nowrap truncate">
                                {controlDetail?.control?.title}
                            </Metric>
                            {severityBadge(controlDetail?.control?.severity)}
                        </Flex>
                        <Flex
                            justifyContent="between"
                            alignItems="start"
                            className="gap-10 w-full"
                        >
                            <Flex
                                flexDirection="col"
                                alignItems="start"
                                className="gap-6 w-2/3"
                            >
                                <div className="group w-[800px] relative flex justify-start">
                                    <Text className="truncate">
                                        {controlDetail?.control?.description}
                                    </Text>
                                    <Card className="absolute w-full z-40 top-0 scale-0 transition-all p-2 group-hover:scale-100">
                                        <Text>
                                            {
                                                controlDetail?.control
                                                    ?.description
                                            }
                                        </Text>
                                    </Card>
                                </div>

                                <Flex justifyContent="start" className="gap-4">
                                    <Button
                                        icon={DocumentTextIcon}
                                        variant="light"
                                        disabled={
                                            (controlDetail?.control
                                                ?.explanation || '') === ''
                                        }
                                        onClick={() => {
                                            setDoc(
                                                controlDetail?.control
                                                    ?.explanation || ''
                                            )
                                            setDocTitle('Detailed Explanation')
                                        }}
                                    >
                                        Show Explanation
                                    </Button>
                                    <div className="border-l h-4 border-gray-300" />
                                    <Button
                                        icon={CommandLineIcon}
                                        variant="light"
                                        onClick={() =>
                                            setModalData(
                                                controlDetail?.control?.query
                                                    ?.queryToExecute || ''
                                            )
                                        }
                                    >
                                        {customizableQuery
                                            ? 'Show Customizable Query'
                                            : 'Show Query'}
                                    </Button>
                                    <div className="border-l h-4 border-gray-300" />
                                    <Button
                                        color="orange"
                                        variant="light"
                                        disabled={reEvaluateButtonLoading}
                                        onClick={() => {
                                            ReEvaluate()
                                        }}
                                    >
                                        <Flex flexDirection="row">
                                            {reEvaluateButtonLoading && (
                                                <Spinner
                                                    className="w-6 h-6"
                                                    color="fill-orange-600"
                                                />
                                            )}
                                            Re-evaluate
                                        </Flex>
                                    </Button>
                                </Flex>
                            </Flex>

                            <Flex
                                flexDirection="col"
                                alignItems="end"
                                justifyContent="start"
                                className="w-1/3 gap-2"
                            >
                                <Flex
                                    flexDirection="row"
                                    justifyContent="start"
                                    className="hover:cursor-pointer max-w-full w-fit bg-gray-200 border-gray-300 rounded-lg border px-1"
                                    onClick={() => {
                                        clipboardCopy(
                                            controlDetail?.control?.id || ''
                                        )
                                    }}
                                >
                                    <Square2StackIcon className="min-w-4 w-4 mr-1" />
                                    <Text className="truncate">
                                        Control ID: {controlDetail?.control?.id}
                                    </Text>
                                </Flex>
                                <Flex
                                    flexDirection="row"
                                    justifyContent="start"
                                    className="max-w-full w-fit bg-gray-200 border-gray-300 rounded-lg border px-1"
                                >
                                    <ClockIcon className="min-w-4 w-4 mr-1" />
                                    <Text className="truncate">
                                        Last updated:{' '}
                                        {(controlDetail?.evaluatedAt || 0) <= 0
                                            ? 'Never'
                                            : dateTimeDisplay(
                                                  controlDetail?.evaluatedAt
                                              )}
                                    </Text>
                                </Flex>

                                {controlDetail?.control?.query?.parameters?.map(
                                    (item) => {
                                        return (
                                            <Flex
                                                flexDirection="row"
                                                justifyContent="start"
                                                className="hover:cursor-pointer max-w-full w-fit bg-gray-200 border-gray-300 rounded-lg border px-1"
                                                onClick={() => {
                                                    navigate(
                                                        `compliance/library/parameters&key=${item.key}`
                                                    )
                                                }}
                                            >
                                                <PencilIcon className="min-w-4 w-4 mr-1" />
                                                <Text className="truncate">
                                                    {item.key}:{' '}
                                                    {parameters?.queryParameters
                                                        ?.filter(
                                                            (p) =>
                                                                p.key ===
                                                                item.key
                                                        )
                                                        .map(
                                                            (p) => p.value || ''
                                                        ) || 'Not defined'}
                                                </Text>
                                            </Flex>
                                        )
                                    }
                                )}
                            </Flex>
                        </Flex>
                    </Flex>
                    <Modal
                        open={!!modalData.length}
                        onClose={() => setModalData('')}
                    >
                        <Title className="font-semibold">Query</Title>
                        <Flex flexDirection="row" alignItems="start">
                            <Card className="my-4">
                                <Editor
                                    onValueChange={() => 1}
                                    highlight={(text) =>
                                        highlight(text, languages.sql, 'sql')
                                    }
                                    value={modalData}
                                    className="w-full bg-white dark:bg-gray-900 dark:text-gray-50 font-mono text-sm"
                                    style={{
                                        minHeight: '200px',
                                    }}
                                    placeholder="-- write your SQL query here"
                                />
                            </Card>
                            <Flex
                                flexDirection="col"
                                justifyContent="start"
                                alignItems="start"
                                className="mt-2 ml-2 w-1/3"
                            >
                                <Text>Customizable Parameters:</Text>
                                {controlDetail?.control?.query?.parameters?.map(
                                    (param) => {
                                        return <Text>- {param.key}</Text>
                                    }
                                )}
                                <Link
                                    to={`/compliance/library`}
                                    className="text-openg-500 cursor-pointer"
                                >
                                    <Text className="text-openg-500">
                                        Click here to change parameters
                                    </Text>
                                </Link>
                            </Flex>
                        </Flex>
                        <Flex>
                            <Button
                                variant="light"
                                icon={DocumentDuplicateIcon}
                                iconPosition="left"
                                onClick={() =>
                                    clipboardCopy(modalData).then(() =>
                                        setNotification({
                                            text: 'Query copied to clipboard',
                                            type: 'info',
                                        })
                                    )
                                }
                            >
                                Copy
                            </Button>
                            <Flex className="w-fit gap-4">
                                <Button
                                    variant="secondary"
                                    onClick={() => {
                                        setQuery(modalData)
                                    }}
                                >
                                    <Link to={`/finder?tab_id=1`}>
                                        Open in Query
                                    </Link>
                                </Button>
                                <Button onClick={() => setModalData('')}>
                                    Close
                                </Button>
                            </Flex>
                        </Flex>
                    </Modal>

                    <Flex justifyContent="start" className="w-full mb-8 gap-6">
                        {costSaving !== 0 && (
                            <SummaryCard
                                title="Estimated Saving Opportunities "
                                metric={costSaving} // TODO-Saleh add saving opportunities
                                isPrice
                                isExact
                            />
                        )}

                        <SummaryCard
                            title={`Compliant ${
                                controlDetail?.resourceType?.resource_name || ''
                            }`}
                            metric={
                                <Flex
                                    flexDirection="row"
                                    justifyContent="start"
                                    alignItems="baseline"
                                    className="gap-3"
                                >
                                    <Metric className="text-emerald-500">
                                        {numberDisplay(
                                            (controlDetail?.totalResourcesCount ||
                                                0) -
                                                (controlDetail?.failedResourcesCount ||
                                                    0),
                                            0
                                        )}
                                    </Metric>
                                    <Text>
                                        of{' '}
                                        {numberDisplay(
                                            controlDetail?.totalResourcesCount ||
                                                0,
                                            0
                                        )}
                                    </Text>
                                </Flex>
                            }
                            isElement
                            onClick={() => {
                                setSelectedTabIndex(0)
                            }}
                            cardClickable
                        />

                        <SummaryCard
                            title={`Non-Compliant ${
                                controlDetail?.resourceType?.resource_name || ''
                            }`}
                            metric={
                                <Flex
                                    flexDirection="row"
                                    justifyContent="start"
                                    alignItems="baseline"
                                    className="gap-3"
                                >
                                    <Metric className="text-rose-500">
                                        {numberDisplay(
                                            controlDetail?.failedResourcesCount ||
                                                0,
                                            0
                                        )}
                                    </Metric>
                                    <Text>
                                        of{' '}
                                        {numberDisplay(
                                            controlDetail?.totalResourcesCount ||
                                                0,
                                            0
                                        )}
                                    </Text>
                                </Flex>
                            }
                            isElement
                            onClick={() => {
                                setSelectedTabIndex(0)
                            }}
                            cardClickable
                        />
                        <SummaryCard
                            // connector={controlDetail?.control?.connector}
                            title={`${
                                controlDetail?.control?.connector?.includes(
                                    SourceType.CloudAWS
                                )
                                    ? 'Impacted AWS Accounts'
                                    : 'Impacted Azure Subscriptions'
                            }`}
                            metric={controlDetail?.totalConnectionCount}
                            onClick={() => {
                                setSelectedTabIndex(1)
                            }}
                            cardClickable
                        />
                    </Flex>

                    {(controlDetail?.control?.manualRemediation &&
                        controlDetail?.control?.manualRemediation.length > 0) ||
                    (controlDetail?.control?.cliRemediation &&
                        controlDetail?.control?.cliRemediation.length > 0) ||
                    (controlDetail?.control?.programmaticRemediation &&
                        controlDetail?.control?.programmaticRemediation.length >
                            0) ||
                    (controlDetail?.control?.guardrailRemediation &&
                        controlDetail?.control?.guardrailRemediation.length >
                            0) ? (
                        <Card className="mb-8 p-8">
                            <Flex
                                justifyContent="start"
                                alignItems="start"
                                className="gap-12"
                            >
                                <Flex
                                    className="w-1/3 h-full"
                                    justifyContent="start"
                                >
                                    <EaseOfSolutionChart
                                        isEmpty
                                        scalability="medium"
                                        complexity="hard"
                                        disruptivity="easy"
                                    />
                                </Flex>

                                <Flex
                                    flexDirection="col"
                                    alignItems="start"
                                    justifyContent="start"
                                    className="h-full w-2/3"
                                >
                                    <DrawerPanel
                                        title={docTitle}
                                        open={doc.length > 0}
                                        onClose={() => setDoc('')}
                                    >
                                        <MarkdownPreview
                                            source={doc}
                                            className="!bg-transparent"
                                            wrapperElement={{
                                                'data-color-mode': 'light',
                                            }}
                                            rehypeRewrite={(
                                                node,
                                                index,
                                                parent
                                            ) => {
                                                if (
                                                    // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                                    // @ts-ignore
                                                    node.tagName === 'a' &&
                                                    parent &&
                                                    /^h(1|2|3|4|5|6)/.test(
                                                        // eslint-disable-next-line @typescript-eslint/ban-ts-comment
                                                        // @ts-ignore
                                                        parent.tagName
                                                    )
                                                ) {
                                                    // eslint-disable-next-line no-param-reassign
                                                    parent.children =
                                                        parent.children.slice(1)
                                                }
                                            }}
                                        />
                                    </DrawerPanel>
                                    <Text className="font-bold mb-4 text-gray-400">
                                        Remediation
                                    </Text>
                                    <Flex className="rounded-lg border border-gray-100 relative">
                                        <Grid
                                            numItems={2}
                                            className="w-full h-full"
                                        >
                                            <Flex
                                                className={
                                                    controlDetail?.control
                                                        ?.manualRemediation &&
                                                    controlDetail?.control
                                                        ?.manualRemediation
                                                        .length
                                                        ? 'cursor-pointer px-6 py-4 h-full gap-3 '
                                                        : 'grayscale opacity-70 px-6 py-4 h-full gap-3'
                                                }
                                                flexDirection="col"
                                                justifyContent="start"
                                                alignItems="start"
                                                onClick={() => {
                                                    if (
                                                        controlDetail?.control
                                                            ?.manualRemediation &&
                                                        controlDetail?.control
                                                            ?.manualRemediation
                                                            .length
                                                    ) {
                                                        setDoc(
                                                            controlDetail
                                                                ?.control
                                                                ?.manualRemediation
                                                        )
                                                        setDocTitle(
                                                            'Manual remediation'
                                                        )
                                                    }
                                                }}
                                            >
                                                <Flex>
                                                    <Flex
                                                        justifyContent="start"
                                                        className="w-fit gap-3"
                                                    >
                                                        <Icon
                                                            icon={BookOpenIcon}
                                                            className="p-0 text-gray-900"
                                                        />
                                                        <Title className="font-semibold">
                                                            Manual
                                                        </Title>
                                                    </Flex>
                                                    <ChevronRightIcon className="w-5 text-openg-500" />
                                                </Flex>
                                                <Text>
                                                    Step by Step Guided solution
                                                    to resolve instances of
                                                    non-compliance
                                                </Text>
                                            </Flex>
                                            <Flex
                                                className={
                                                    controlDetail?.control
                                                        ?.cliRemediation &&
                                                    controlDetail?.control
                                                        ?.cliRemediation.length
                                                        ? 'cursor-pointer px-6 py-4 h-full gap-3 '
                                                        : 'grayscale opacity-70 px-6 py-4 h-full gap-3'
                                                }
                                                flexDirection="col"
                                                justifyContent="start"
                                                alignItems="start"
                                                onClick={() => {
                                                    if (
                                                        controlDetail?.control
                                                            ?.cliRemediation &&
                                                        controlDetail?.control
                                                            ?.cliRemediation
                                                            .length
                                                    ) {
                                                        setDoc(
                                                            controlDetail
                                                                ?.control
                                                                ?.cliRemediation
                                                        )
                                                        setDocTitle(
                                                            'Command line (CLI) remediation'
                                                        )
                                                    }
                                                }}
                                            >
                                                <Flex>
                                                    <Flex
                                                        justifyContent="start"
                                                        className="w-fit gap-3"
                                                    >
                                                        <Icon
                                                            icon={
                                                                CommandLineIcon
                                                            }
                                                            className="p-0 text-gray-900"
                                                        />
                                                        <Title className="font-semibold">
                                                            Command line (CLI)
                                                        </Title>
                                                    </Flex>
                                                    <ChevronRightIcon className="w-5 text-openg-500" />
                                                </Flex>
                                                <Text>
                                                    Guided steps to resolve the
                                                    issue utilizing CLI
                                                </Text>
                                            </Flex>
                                            <Flex
                                                className={
                                                    controlDetail?.control
                                                        ?.guardrailRemediation &&
                                                    controlDetail?.control
                                                        ?.guardrailRemediation
                                                        .length
                                                        ? 'cursor-pointer px-6 py-4 h-full gap-3 '
                                                        : 'grayscale opacity-70 px-6 py-4 h-full gap-3'
                                                }
                                                flexDirection="col"
                                                justifyContent="start"
                                                alignItems="start"
                                                onClick={() => {
                                                    if (
                                                        controlDetail?.control
                                                            ?.guardrailRemediation &&
                                                        controlDetail?.control
                                                            ?.guardrailRemediation
                                                            .length
                                                    ) {
                                                        setDoc(
                                                            controlDetail
                                                                ?.control
                                                                ?.guardrailRemediation
                                                        )
                                                        setDocTitle(
                                                            'Guard rails remediation'
                                                        )
                                                    }
                                                }}
                                            >
                                                <Flex>
                                                    <Flex
                                                        justifyContent="start"
                                                        className="w-fit gap-3"
                                                    >
                                                        <Icon
                                                            icon={Cog8ToothIcon}
                                                            className="p-0 text-gray-900"
                                                        />
                                                        <Title className="font-semibold">
                                                            Guard rails
                                                        </Title>
                                                    </Flex>
                                                    <ChevronRightIcon className="w-5 text-openg-500" />
                                                </Flex>
                                                <Text>
                                                    Resolve and ensure
                                                    compliance, at scale
                                                    utilizing solutions where
                                                    possible
                                                </Text>
                                            </Flex>
                                            <Flex
                                                className={
                                                    controlDetail?.control
                                                        ?.programmaticRemediation &&
                                                    controlDetail?.control
                                                        ?.programmaticRemediation
                                                        .length
                                                        ? 'cursor-pointer px-6 py-4 h-full gap-3 '
                                                        : 'grayscale opacity-70 px-6 py-4 h-full gap-3'
                                                }
                                                flexDirection="col"
                                                justifyContent="start"
                                                alignItems="start"
                                                onClick={() => {
                                                    if (
                                                        controlDetail?.control
                                                            ?.programmaticRemediation &&
                                                        controlDetail?.control
                                                            ?.programmaticRemediation
                                                            .length
                                                    ) {
                                                        setDoc(
                                                            controlDetail
                                                                ?.control
                                                                ?.programmaticRemediation
                                                        )
                                                        setDocTitle(
                                                            'Programmatic remediation'
                                                        )
                                                    }
                                                }}
                                            >
                                                <Flex>
                                                    <Flex
                                                        justifyContent="start"
                                                        className="w-fit gap-3"
                                                    >
                                                        <Icon
                                                            icon={
                                                                CodeBracketIcon
                                                            }
                                                            className="p-0 text-gray-900"
                                                        />
                                                        <Title className="font-semibold">
                                                            Programmatic
                                                        </Title>
                                                    </Flex>
                                                    <ChevronRightIcon className="w-5 text-openg-500" />
                                                </Flex>
                                                <Text>
                                                    Scripts that help you
                                                    resolve the issue, at scale
                                                </Text>
                                            </Flex>
                                        </Grid>
                                        <div className="border-t border-gray-100 w-full absolute top-1/2" />
                                        <div className="border-l border-gray-100 h-full absolute left-1/2" />
                                    </Flex>
                                </Flex>
                            </Flex>
                        </Card>
                    ) : null}

                    {!hideTabs && (
                        <TabGroup
                            key={`tabs-${selectedTabIndex}`}
                            defaultIndex={selectedTabIndex}
                            tabIndex={selectedTabIndex}
                            onIndexChange={setSelectedTabIndex}
                        >
                            <Flex
                                flexDirection="row"
                                justifyContent="between"
                                className="mb-2"
                            >
                                <div className="w-fit">
                                    <TabList>
                                        <Tab>Impacted resources</Tab>
                                        <Tab>Impacted accounts</Tab>
                                        {/* <Tab>Findings</Tab> */}
                                    </TabList>
                                </div>
                                <Flex flexDirection="row" className="w-fit">
                                    <Text className="mr-2 w-fit">
                                        Confomance Status filter:
                                    </Text>
                                    <TabGroup
                                        tabIndex={conformanceFilterIdx()}
                                        className="w-fit"
                                        onIndexChange={(tabIndex) => {
                                            switch (tabIndex) {
                                                case 1:
                                                    setConformanceFilter([
                                                        GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus.ConformanceStatusFailed,
                                                    ])
                                                    break
                                                case 2:
                                                    setConformanceFilter([
                                                        GithubComKaytuIoKaytuEnginePkgComplianceApiConformanceStatus.ConformanceStatusPassed,
                                                    ])
                                                    break
                                                default:
                                                    setConformanceFilter(
                                                        undefined
                                                    )
                                            }
                                        }}
                                    >
                                        <TabList variant="solid">
                                            <Tab value="1">All</Tab>
                                            <Tab value="2">Failed</Tab>
                                            <Tab value="3">Passed</Tab>
                                        </TabList>
                                    </TabGroup>
                                </Flex>
                            </Flex>
                            <TabPanels>
                                <TabPanel>
                                    {selectedTabIndex === 0 && (
                                        <ImpactedResources
                                            controlId={
                                                controlDetail?.control?.id || ''
                                            }
                                            conformanceFilter={
                                                conformanceFilter
                                            }
                                            linkPrefix={`/score/categories/`}
                                            isCostOptimization={
                                                costSaving !== 0
                                            }
                                        />
                                    )}
                                </TabPanel>
                                <TabPanel>
                                    {selectedTabIndex === 1 && (
                                        <ImpactedAccounts
                                            controlId={
                                                controlDetail?.control?.id
                                            }
                                        />
                                    )}
                                </TabPanel>
                                {/* <TabPanel>
                                    {selectedTabIndex === 2 && (
                                        <ControlFindings
                                            onlyFailed={onlyFailed}
                                            controlId={
                                                controlDetail?.control?.id
                                            }
                                        />
                                    )}
                                </TabPanel> */}
                            </TabPanels>
                        </TabGroup>
                    )}
                </>
            )}
        </>
    )
}
