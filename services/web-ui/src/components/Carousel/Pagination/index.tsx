import {
    ArrowLongLeftIcon,
    ArrowLongRightIcon,
} from '@heroicons/react/20/solid'
import { Button, Flex, Text } from '@tremor/react'

type IpProps = {
    onClickNext?: any
    onClickPrevious?: any
    pageCount?: any
    currentPage?: any
}
export default function Pagination({
    onClickNext,
    onClickPrevious,
    pageCount,
    currentPage,
}: IpProps) {
    return (
        <Flex className="w-full border-t border-gray-200 pt-2 mt-4">
            <Button
                disabled={currentPage === 1}
                variant="light"
                icon={ArrowLongLeftIcon}
                onClick={onClickPrevious}
            >
                Previous
            </Button>
            <Text className="hidden md:flex">
                Page {currentPage} of {pageCount}
            </Text>
            <Button
                disabled={currentPage >= pageCount}
                variant="light"
                onClick={onClickNext}
                icon={ArrowLongRightIcon}
                iconPosition="right"
            >
                Next
            </Button>
        </Flex>
    )
}
