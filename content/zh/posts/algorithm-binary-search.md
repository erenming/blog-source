---
title: "[刷题总结]二分搜索"
date: 2022-11-04T13:27:25+08:00
draft: false
Tags: [Algorithm]
---

二分搜索算法，又称折半搜索，是一种通过寻找中点并不断减半搜索范围的算法，时间复杂度为`O(logN)`，其算法流程大致如下所述：

1. 首先，其必须作用于有序数组上，找到**中点**并与目标值进行比较
2. 若目标值比中点小，则将搜索范围缩小为左半数组；反之则将搜索范围缩小为右半数组
3. 重复1，2步骤，直到终点与目标值相等

二分搜索的算法流程看起来很简单，然而实际的题目可能会有各种变化和细节，因此写好一个二分搜索算法并不简单。

在我看来，二分搜索的关键点在于问题的**划分点**以及**子集的选择**，而这两块也最容易产生变体：

- 有序性变体：目标序列并非严格有序，或者说有序性需要你自行构造

- 二分选择变体：左右子集的选择算法往往不尽相同

此外，我们需要特别**注意细节**，因此编写中应当尽量使用`else if`而非`else`，力求覆盖到每种case

# 框架

```go
func search(nums []int, target int) int {
  lo, hi := 0, n-1
  for less(lo, hi) {
    mid := getmid(lo, hi)
    if equalcase(mid, target) {
      // 1. 先考虑相等情况
    } else if lesscase(target, mid) {
      // 2. 再考虑其他情况
    } else if ...
  }
}
```

# 题目详解

## [33. 搜索旋转排序数组](https://leetcode.cn/problems/search-in-rotated-sorted-array/description/)

**有序性变体**

此题题意为在一个旋转的有序数组中搜索目标值的下标，此题的关键在于**选边**，因为有且仅有左边严格有序或右边严格有序。

```go
func search(nums []int, target int) int {
	n := len(nums)
	lo, hi := 0, n-1
	for lo <= hi {
		mid := lo + (hi-lo)/2
		if nums[mid] == target {
			return mid
		} else if nums[lo] <= nums[mid] { // 左边严格有序
			if nums[lo] <= target && target < nums[mid] { // 目标值在左边序列
				hi = mid - 1 // 搜索左边序列
			} else {
				lo = mid + 1 // 否则搜索右边序列
			}
		} else { // 右边严格有序
			if nums[hi] >= target && target > nums[mid] {  // 目标值在右边序列
				lo = mid + 1 // 搜素右边序列
			} else {
				hi = mid - 1 // 否则搜索左边序列
			}
		}
	}
	return -1
}

```

## [162. 寻找峰值](https://leetcode.cn/problems/find-peak-element/description/)

此题存在有序性变体，二分选择条件变体。

有序性：局部有序

二分选择：通过比较相邻两元素的大小，来判断峰值所在的子序列

```go
func findPeakElement(nums []int) int {
	n := len(nums)
	lo, hi := 0, n-1
	for lo < hi {
		mid := lo + (hi-lo)/2
		if nums[mid] <= nums[mid+1] { // 说明是上坡，峰值在右侧
			lo = mid + 1
		} else { // 说明是下坡，峰值在左侧
			hi = mid
		}
	}
	return lo
}
```

## 	[209. 长度最小的子数组](https://leetcode.cn/problems/minimum-size-subarray-sum/description/)

此题存在有序性变体。nums为正整数序列，因此其前缀和序列为单调非减序列，我们构造前缀和序列sums。

target满足当j > i 时，sums[j] - sums[i] = target

```go

func minSubArrayLen(target int, nums []int) int {
	n := len(nums)
	// sums为前缀和序列
  // sums[i]表示长度为i的前缀和
	sums := make([]int, n+1)
	for i := 1; i < len(sums); i++ {
		sums[i] = nums[i-1] + sums[i-1]
	}
	res := math.MaxInt64
	for i := 0; i < n+1; i++ {
		need := target + sums[i] // 根据公式，我们需要通过二分法查找sums[idx]=need的下标
		lo, hi := i+1, n
		k := i
		for lo <= hi {
			mid := lo + (hi-lo)/2
			if need == sums[mid] { // 等于need时，由于是查找最小序列，因此取左边序列
				k = mid // 更新下标k
				hi = mid - 1
			} else if need < sums[mid] { // 小于need时，查找左边序列
				k = mid // 更新下标k
				hi = mid - 1
			} else { // 大于need时，查找右边序列
				lo = mid + 1
			}
		}
		if k > i { // 找到对应下标时，更新res
			res = min(res, k-i)
		}
	}
	if res == math.MaxInt64 {
		return 0
	}
	return res
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

```

## [287. 寻找重复数](https://leetcode.cn/problems/find-the-duplicate-number/description/)

此题需要对鸽笼原理有所了解，同时存在有序性变体和二分选择变体。根据鸽笼原理，我们可以知道，对于x，1<=x<=n，若nums中小于等于x的数目大于x，则重复的数一定在左边子序列中。

这是因为nums中的数除了重复数都跟[1...n]中的数一一对应，重复数在左侧则会使得中位数右移，重复数在右侧则会使中位数左移。

对于[1..n]序列来说，我们分析可得:

> 对于重复数位于左侧：[1, 2, 2, 3, 4, 5]，mid为3，右移了一位
>
> 对于重复数位于右侧：[1, 2, 3, 4, 5, 5]，mid为3，左移了一位

```go
func findDuplicate(nums []int) int {
	lo, hi := 1, len(nums)-1 // 这里搜索的是[1..n]的有序序列
	for lo < hi {
		mid := lo + (hi-lo)/2
		cnt := 0
		for _, item := range nums {
			if item <= mid {
				cnt++
			}
		}
		if cnt > mid {
			hi = mid
		} else if cnt == mid { // 相等时，重复数在右侧
			lo = mid+1
		} else {
			lo = mid+1
		}
	}
	return lo
}
```

## [540. 有序数组中的单一元素](https://leetcode.cn/problems/single-element-in-a-sorted-array/description/)

此题存在二分选择变体，根据有序序列的下标性质

- 若mid为偶数，且nums[mid] == nums[mid+1]，则单一元素在右边，否则在左边
- 若mid为奇数，且nums[mid] == nums[mid+1]，则单一元素在左边，否则在右边

```go

func singleNonDuplicate(nums []int) int {
	n := len(nums)
	lo, hi := 0, n-1
	for lo < hi {
		mid := lo + (hi-lo)/2
		if nums[mid] == nums[mid+1] {
			if mid%2 == 1 {
				hi = mid - 1
			} else {
				lo = mid
			}
		} else {
			if mid%2 == 1 {
				lo = mid + 1
			} else {
				hi = mid
			}
		}
	}
	return nums[hi]
}
```



# 参考

- https://en.wikipedia.org/wiki/Binary_search_algorithm
- https://www.cnblogs.com/kyoner/p/11080078.html
