import { Flex, Grid, GridProps } from '@tremor/react'
import { Children, ReactNode, useEffect, useMemo, useState } from 'react'
import Pagination from './Pagination'

interface IProps {
    children: ReactNode
    pageSize?: number
}

export default function Carousel({ children, pageSize = 9 }: IProps) {
    const [activeSlide, setActiveSlider] = useState<number>(1)

    const itemsCount = useMemo(
        () => Children.toArray(children).length,
        [children]
    )

    const renderItems = (slideNumber = 1) => {
        const items = Children.toArray(children).slice(
            (slideNumber - 1) * pageSize,
            slideNumber * pageSize
        )
        return Children.map(items, (child) => child)
    }

    const handlePageChange = (page: number) => {
        const pageCount = Math.ceil(itemsCount / pageSize)
        if (activeSlide + page > pageCount) {
            setActiveSlider(activeSlide)
        } else if (activeSlide + page <= 0) {
            setActiveSlider(1)
        } else {
            setActiveSlider(activeSlide + page)
        }
    }

    useEffect(() => {
        setActiveSlider(1)
    }, [children])

    return (
        <Flex flexDirection="col" className="min-h-[765px]">
            <Grid numItems={3} className="w-full gap-4">
                {renderItems(activeSlide)}
            </Grid>
            <Pagination
                onClickNext={() => handlePageChange(1)}
                onClickPrevious={() => handlePageChange(-1)}
                currentPage={activeSlide}
                pageCount={Math.ceil(itemsCount / pageSize)}
            />
        </Flex>
    )
}
