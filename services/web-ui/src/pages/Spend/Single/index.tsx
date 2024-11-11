import { useParams } from 'react-router-dom'
import NotFound from '../../Errors'
import SingleSpendConnection from './SingleConnection'
import SingleSpendMetric from './SingleMetric'
import TopHeader from '../../../components/Layout/Header'
import {
    defaultSpendTime,
    useUrlDateRangeState,
} from '../../../utilities/urlstate'

export default function SingleSpend() {
    const { ws } = useParams()
    const { value: activeTimeRange } = useUrlDateRangeState(
        defaultSpendTime(ws || '')
    )
    const { id, metric } = useParams()
    const urlParams = window.location.pathname.split('/')
    if (urlParams[1] === 'ws') {
        urlParams.shift()
    }

    const idGenerator = () => {
        if (metric) {
            if (urlParams[urlParams.length - 1].startsWith('account_')) {
                return metric.replace('account_', '')
            }
            if (urlParams[urlParams.length - 1].startsWith('metric_')) {
                return metric.replace('metric_', '')
            }
            return undefined
        }
        if (id) {
            if (urlParams[urlParams.length - 1].startsWith('account_')) {
                return id.replace('account_', '')
            }
            if (urlParams[urlParams.length - 1].startsWith('metric_')) {
                return id.replace('metric_', '')
            }
            return undefined
        }
        return undefined
    }

    const renderPage = () => {
        if (urlParams[urlParams.length - 1].startsWith('account_')) {
            return (
                <>
                    <TopHeader
                        breadCrumb={['Cloud account spend detail']}
                        datePickerDefault={defaultSpendTime(ws || '')}
                        supportedFilters={['Date']}
                        initialFilters={['Date']}
                    />
                    <SingleSpendConnection
                        activeTimeRange={activeTimeRange}
                        id={idGenerator()}
                    />
                </>
            )
        }
        if (urlParams[urlParams.length - 1].startsWith('metric_')) {
            return (
                <>
                    <TopHeader
                        breadCrumb={['Metric spend detail']}
                        datePickerDefault={defaultSpendTime(ws || '')}
                        supportedFilters={[
                            'Date',
                            'Cloud Account',
                            'Connector',
                        ]}
                        initialFilters={['Date']}
                    />
                    <SingleSpendMetric
                        activeTimeRange={activeTimeRange}
                        metricId={idGenerator()}
                    />
                </>
            )
        }
        return <NotFound />
    }

    return renderPage()
}
