import {
    CheckCircleIcon,
    XCircleIcon,
    CloudIcon,
    CalendarIcon,
    DocumentCheckIcon,
    ShieldCheckIcon,
    PuzzlePieceIcon,
    EyeIcon,
    CubeIcon,
} from '@heroicons/react/24/outline'
import {
    AWSIcon,
    AzureIcon,
    CloudConnect,
    Lifecycle,
    SeverityIcon,
    TagIcon,
    DocumentBadge,
    EntraIDIcon,
    AWSAzureIcon,
} from '../../../icons/icons'
import CheckboxSelector, { CheckboxItem } from '../CheckboxSelector'
import RadioSelector, { RadioItem } from '../RadioSelector'
import { DateRange } from '../../../utilities/urlstate'
import DateSelector, { renderDateText } from '../DateSelector'
import { DateSelectorOptions } from '../ConditionSelector/DateConditionSelector'

export function ConformanceFilter(
    selectedValue: string,
    onValueSelected: (sv: string) => void,
    onReset: () => void
) {
    const conformanceValues: RadioItem[] = [
        { title: 'All', value: 'all' },
        {
            title: 'Failed',
            iconAlt: <XCircleIcon className="text-rose-500 w-5 mr-1" />,
            value: 'failed',
        },
        {
            title: 'Passed',
            iconAlt: <CheckCircleIcon className="text-emerald-500 w-5 mr-1" />,
            value: 'passed',
        },
    ]

    return {
        title: 'Conformance Status',
        icon: CheckCircleIcon,
        itemsTitles: conformanceValues
            .filter((i) => selectedValue === i.value)
            .map((i) => i.title),
        isValueChanged: true,
        selector: (
            <RadioSelector
                title="Conformance Status"
                radioItems={conformanceValues}
                selectedValue={selectedValue}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function ConnectorFilter(
    selectedValue: string,
    isValueChanged: boolean,
    onValueSelected: (sv: string) => void,
    onRemove: () => void,
    onReset: () => void
) {
    const connectorValues: RadioItem[] = [
        {
            title: 'All',
            value: '',
        },
        {
            title: 'AWS',
            icon: (
                <img
                    src={AWSIcon}
                    className="w-5 mr-2 rounded-full"
                    alt="aws"
                />
            ),
            value: 'AWS',
        },
        {
            title: 'Azure',
            icon: (
                <img
                    src={AzureIcon}
                    className="w-5 mr-2 rounded-full"
                    alt="azure"
                />
            ),
            value: 'Azure',
        },
        {
            title: 'Entra ID',
            icon: (
                <img
                    src={EntraIDIcon}
                    className="w-5 mr-2 rounded-full"
                    alt="entra id"
                />
            ),
            value: 'EntraID',
        },
    ]

    return {
        title: 'Connector',
        icon: CloudConnect,
        itemsTitles: connectorValues
            .filter((i) => selectedValue === i.value)
            .map((i) => i.title),
        isValueChanged,
        selector: (
            <RadioSelector
                title="Connector"
                radioItems={connectorValues}
                selectedValue={selectedValue}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function LifecycleFilter(
    selectedValue: string,
    isValueChanged: boolean,
    onValueSelected: (sv: string) => void,
    onRemove: () => void,
    onReset: () => void
) {
    const lifecycleValues: RadioItem[] = [
        { title: 'All', value: 'all' },
        { title: 'Active', value: 'active' },
        { title: 'Archived', value: 'archived' },
    ]

    return {
        title: 'Lifecycle',
        icon: Lifecycle,
        itemsTitles: lifecycleValues
            .filter((i) => selectedValue === i.value)
            .map((i) => i.title),
        isValueChanged,
        selector: (
            <RadioSelector
                title="Lifecycle"
                radioItems={lifecycleValues}
                selectedValue={selectedValue}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function SeverityFilter(
    selectedValues: string[],
    isValueChanged: boolean,
    onValueSelected: (sv: string) => void,
    onRemove: () => void,
    onReset: () => void
) {
    const severityValues: CheckboxItem[] = [
        {
            title: 'Critical',
            iconAlt: (
                <div
                    className="h-4 w-2 rounded-sm mr-1.5"
                    style={{ backgroundColor: '#6E120B' }}
                />
            ),
            value: 'critical',
        },
        {
            title: 'High',
            iconAlt: (
                <div
                    className="h-4 w-2 rounded-sm mr-1.5"
                    style={{ backgroundColor: '#CA2B1D' }}
                />
            ),
            value: 'high',
        },
        {
            title: 'Medium',
            iconAlt: (
                <div
                    className="h-4 w-2 rounded-sm mr-1.5"
                    style={{ backgroundColor: '#EE9235' }}
                />
            ),
            value: 'medium',
        },
        {
            title: 'Low',
            iconAlt: (
                <div
                    className="h-4 w-2 rounded-sm mr-1.5"
                    style={{ backgroundColor: '#F4C744' }}
                />
            ),
            value: 'low',
        },
        {
            title: 'None',
            iconAlt: (
                <div
                    className="h-4 w-2 rounded-sm mr-1.5"
                    style={{ backgroundColor: '#9BA2AE' }}
                />
            ),
            value: 'none',
        },
    ]

    return {
        title: 'Severity',
        icon: SeverityIcon,
        itemsTitles: severityValues
            .filter((i) => selectedValues.includes(i.value))
            .map((i) => i.title),
        isValueChanged,
        selector: (
            <CheckboxSelector
                title="Severity"
                checkboxItems={severityValues}
                selectedValues={selectedValues}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function CloudAccountFilter(
    items: CheckboxItem[],
    onValueSelected: (sv: string) => void,
    selectedValues: string[],
    isValueChanged: boolean,
    onRemove: () => void,
    onReset: () => void,
    onSearch: (i: string) => void
) {
    return {
        title: 'Cloud Account',
        icon: CloudIcon,
        itemsTitles: items
            .filter((item) => selectedValues.includes(item.value))
            .map((item) => item.title),
        isValueChanged,
        selector: (
            <CheckboxSelector
                title="Cloud Account"
                checkboxItems={items}
                selectedValues={selectedValues}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
                onSearch={onSearch}
            />
        ),
    }
}

export function ServiceNameFilter(
    items: CheckboxItem[],
    onValueSelected: (sv: string) => void,
    selectedValues: string[],
    isValueChanged: boolean,
    onRemove: () => void,
    onReset: () => void
) {
    return {
        title: 'Service Name',
        icon: DocumentBadge,
        itemsTitles: items
            .filter((item) => selectedValues.includes(item.value))
            .map((item) => item.title),
        isValueChanged,
        selector: (
            <CheckboxSelector
                title="Service Name"
                checkboxItems={items}
                selectedValues={selectedValues}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function ScoreTagFilter(
    items: CheckboxItem[],
    onValueSelected: (sv: string) => void,
    selectedValues: string[],
    isValueChanged: boolean,
    onRemove: () => void,
    onReset: () => void
) {
    return {
        title: 'Tag',
        icon: TagIcon,
        itemsTitles: items
            .filter((item) => selectedValues.includes(item.value))
            .map((item) => item.title),
        isValueChanged,
        selector: (
            <CheckboxSelector
                title="Tag"
                checkboxItems={items}
                selectedValues={selectedValues}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function DateFilter(
    value: DateRange,
    onValueChange: (i: DateRange) => void,
    selectedCondition: DateSelectorOptions,
    onConditionChange: (i: DateSelectorOptions) => void
) {
    return {
        title: 'Date',
        icon: CalendarIcon,
        itemsTitles: [renderDateText(value.start, value.end)],
        isValueChanged: true,
        selector: (
            <DateSelector
                title="Date"
                value={value}
                supportedConditions={['isBetween', 'isRelative']}
                selectedCondition={selectedCondition}
                onValueChanged={onValueChange}
                onConditionChange={onConditionChange}
            />
        ),
    }
}

export function ProductFilter(onRemove: () => void) {
    return {
        title: 'Product',
        icon: DocumentBadge,
        itemsTitles: ['All'],
        isValueChanged: true,
        selector: (
            <RadioSelector
                title="Product"
                radioItems={[
                    {
                        title: 'All',
                        value: '',
                    },
                ]}
                selectedValue=""
                onItemSelected={(t) => 1}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function EnvironmentFilter(onRemove: () => void) {
    return {
        title: 'Environment',
        icon: DocumentBadge,
        itemsTitles: ['All'],
        isValueChanged: true,
        selector: (
            <RadioSelector
                title="Environment"
                radioItems={[
                    {
                        title: 'All',
                        value: '',
                    },
                ]}
                selectedValue=""
                onItemSelected={(t) => 1}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onConditionChange={() => ''}
            />
        ),
    }
}
export function ScoreCategory(
    selectedValue: string,
    isValueChanged: boolean,
    onValueSelected: (sv: string) => void,
    onRemove: () => void,
    onReset: () => void
) {
    const categoryValues: RadioItem[] = [
        { title: 'All SCORE Insights', value: '' },
        {
            title: 'Security',
            value: 'security',
        },
        {
            title: 'Cost Optimization',
            value: 'cost_optimization',
        },
        {
            title: 'Operational Excellence',
            value: 'operational_excellence',
        },
        {
            title: 'Reliability',
            value: 'reliability',
        },
        {
            title: 'Efficiency',
            value: 'efficiency',
        },
    ]

    return {
        title: 'Score Category',
        icon: PuzzlePieceIcon,
        itemsTitles: categoryValues
            .filter((i) => selectedValue === i.value)
            .map((i) => i.title),
        isValueChanged,
        selector: (
            <RadioSelector
                title="Score Category"
                radioItems={categoryValues}
                selectedValue={selectedValue}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function BenchmarkStateFilter(
    selectedValue: string,
    isValueChanged: boolean,
    onValueSelected: (sv: any) => void,
    onReset: () => void
) {
    const benchmarkStateValues: RadioItem[] = [
        { title: 'All', value: '' },
        {
            title: 'Active',
            value: 'active',
        },
        {
            title: 'Not Active',
            value: 'notactive',
        },
    ]

    return {
        title: 'State',
        icon: ShieldCheckIcon,
        itemsTitles: benchmarkStateValues
            .filter((i) => selectedValue === i.value)
            .map((i) => i.title),
        isValueChanged,
        selector: (
            <RadioSelector
                title="State"
                radioItems={benchmarkStateValues}
                selectedValue={selectedValue}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function BenchmarkAuditTrackingFilter(
    selectedValue: string,
    isValueChanged: boolean,
    onValueSelected: (sv: any) => void,
    onRemove: () => void,
    onReset: () => void
) {
    const benchmarkAuditTrackingValues: RadioItem[] = [
        {
            title: 'Enabled',
            value: 'enabled',
        },
        {
            title: 'Disabled',
            value: 'disabled',
        },
    ]

    return {
        title: 'Audit Tracking',
        icon: EyeIcon,
        itemsTitles: benchmarkAuditTrackingValues
            .filter((i) => selectedValue === i.value)
            .map((i) => i.title),
        isValueChanged,
        selector: (
            <RadioSelector
                title="Audit Tracking"
                radioItems={benchmarkAuditTrackingValues}
                selectedValue={selectedValue}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}

export function BenchmarkFrameworkFilter(
    selectedValues: string[],
    isValueChanged: boolean,
    onValueSelected: (sv: string) => void,
    onRemove: () => void,
    onReset: () => void
) {
    const benchmarkFrameworkValues: CheckboxItem[] = [
        {
            title: 'Service Best Practices',
            value: 'bestpractices',
        },
        {
            title: 'SOC',
            value: 'soc',
        },
        {
            title: 'CISA',
            value: 'cisa',
        },
        {
            title: 'DoD',
            value: 'dod',
        },
        {
            title: 'Privacy',
            value: 'privacy',
        },
    ]

    return {
        title: 'Framework',
        icon: CubeIcon,
        itemsTitles: benchmarkFrameworkValues
            .filter((i) => selectedValues.includes(i.value))
            .map((i) => i.title),
        isValueChanged,
        selector: (
            <CheckboxSelector
                title="Framework"
                checkboxItems={benchmarkFrameworkValues}
                selectedValues={selectedValues}
                onItemSelected={(t) => onValueSelected(t.value)}
                supportedConditions={['is']}
                selectedCondition="is"
                onRemove={onRemove}
                onReset={onReset}
                onConditionChange={() => ''}
            />
        ),
    }
}
