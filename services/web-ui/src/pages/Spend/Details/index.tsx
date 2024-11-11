import { Tab, TabGroup, TabList } from '@tremor/react'
import {
    useLocation,
    useNavigate,
    useParams,
    useSearchParams,
} from 'react-router-dom'
import { useEffect, useState } from 'react'
import { useAtomValue } from 'jotai'
import { checkGranularity } from '../../../utilities/dateComparator'
import TopHeader from '../../../components/Layout/Header'
import {
    defaultSpendTime,
    searchAtom,
    useUrlDateRangeState,
} from '../../../utilities/urlstate'

export default function SpendDetails() {
    const { ws } = useParams()
    const navigate = useNavigate()
    const searchParams = useAtomValue(searchAtom)
    const { value: activeTimeRange } = useUrlDateRangeState(
        defaultSpendTime(ws || '')
    )

    const [selectedTab, setSelectedTab] = useState(0)
    const tabs = useLocation().hash
    useEffect(() => {
        switch (tabs) {
            case '#metrics':
                setSelectedTab(0)
                break
            case '#cloud-accounts':
                setSelectedTab(1)
                break
            default:
                setSelectedTab(0)
                break
        }
    }, [tabs])
    const [selectedGranularity, setSelectedGranularity] = useState<
        'monthly' | 'daily'
    >(
        checkGranularity(activeTimeRange.start, activeTimeRange.end).monthly
            ? 'monthly'
            : 'daily'
    )
    useEffect(() => {
        setSelectedGranularity(
            checkGranularity(activeTimeRange.start, activeTimeRange.end).monthly
                ? 'monthly'
                : 'daily'
        )
    }, [activeTimeRange])

    return (
        <>
            <TopHeader
                breadCrumb={['Spend detail']}
                supportedFilters={['Date', 'Cloud Account', 'Connector']}
                initialFilters={['Date']}
                datePickerDefault={defaultSpendTime(ws || '')}
            />
            <TabGroup index={selectedTab} onIndexChange={setSelectedTab}>
                <TabList className="mb-3">
                    <Tab onClick={() => navigate(`#metrics?${searchParams}`)}>
                        Metrics
                    </Tab>
                    <Tab
                        onClick={() =>
                            navigate(`#cloud-accounts?${searchParams}`)
                        }
                    >
                        Cloud accounts
                    </Tab>
                </TabList>
                {/* <TabPanels>
                    <TabPanel>
                        <Metrics
                            activeTimeRange={activeTimeRange}
                            connections={selectedConnections}
                            selectedGranularity={selectedGranularity}
                            onGranularityChange={setSelectedGranularity}
                            isSummary={tabs === '#category'}
                        />
                    </TabPanel>
                    <TabPanel>
                        <CloudAccounts
                            activeTimeRange={activeTimeRange}
                            connections={selectedConnections}
                            selectedGranularity={selectedGranularity}
                            onGranularityChange={setSelectedGranularity}
                        />
                    </TabPanel>
                </TabPanels> */}
            </TabGroup>
        </>
    )
}
