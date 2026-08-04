package admin

import (
	"sort"
	"sync"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func (h *AccountHandler) batchDeleteAccounts(c *gin.Context) {
	var req struct {
		AccountIDs []int64 `json:"account_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request: "+err.Error())
		return
	}
	accountIDs := normalizeInt64IDList(req.AccountIDs)
	if len(accountIDs) == 0 {
		response.BadRequest(c, "account_ids is required")
		return
	}

	accounts, err := h.adminService.GetAccountsByIDs(c.Request.Context(), accountIDs)
	if err != nil {
		response.ErrorFrom(c, err)
		return
	}
	accountsByID := make(map[int64]*service.Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			accountsByID[account.ID] = account
		}
	}
	requested := make(map[int64]struct{}, len(accountIDs))
	for _, id := range accountIDs {
		requested[id] = struct{}{}
	}

	rootIDs := make([]int64, 0, len(accountIDs))
	dependents := make(map[int64][]int64)
	failedIDs := make([]int64, 0)
	failedErrors := make([]gin.H, 0)
	for _, id := range accountIDs {
		account := accountsByID[id]
		if account == nil {
			failedIDs = append(failedIDs, id)
			failedErrors = append(failedErrors, gin.H{"account_id": id, "error": "account not found"})
			continue
		}
		root := id
		seen := map[int64]struct{}{id: {}}
		for {
			current := accountsByID[root]
			if current == nil || current.ParentAccountID == nil {
				break
			}
			parent := *current.ParentAccountID
			if _, ok := requested[parent]; !ok {
				break
			}
			if _, ok := accountsByID[parent]; !ok {
				break
			}
			if _, ok := seen[parent]; ok {
				root = id
				break
			}
			seen[parent] = struct{}{}
			root = parent
		}
		if root == id {
			rootIDs = append(rootIDs, id)
		} else {
			dependents[root] = append(dependents[root], id)
		}
	}

	successIDs := make([]int64, 0, len(accountIDs))
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(c.Request.Context())
	g.SetLimit(5)
	for _, root := range rootIDs {
		root := root
		g.Go(func() error {
			err := h.adminService.DeleteAccount(gctx, root)
			mu.Lock()
			defer mu.Unlock()
			ids := append([]int64{root}, dependents[root]...)
			if err != nil {
				for _, id := range ids {
					failedIDs = append(failedIDs, id)
					failedErrors = append(failedErrors, gin.H{"account_id": id, "error": err.Error()})
				}
				return nil
			}
			successIDs = append(successIDs, ids...)
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		response.ErrorFrom(c, err)
		return
	}
	sort.Slice(successIDs, func(i, j int) bool { return successIDs[i] < successIDs[j] })
	sort.Slice(failedIDs, func(i, j int) bool { return failedIDs[i] < failedIDs[j] })
	sort.Slice(failedErrors, func(i, j int) bool {
		return failedErrors[i]["account_id"].(int64) < failedErrors[j]["account_id"].(int64)
	})
	response.Success(c, gin.H{
		"total": len(accountIDs), "success": len(successIDs), "failed": len(failedIDs),
		"success_ids": successIDs, "failed_ids": failedIDs, "errors": failedErrors,
	})
}
