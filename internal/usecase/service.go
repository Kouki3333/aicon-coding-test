package usecase

import (
	"context"
	"fmt"

	"Aicon-assignment/internal/domain/entity"
	domainErrors "Aicon-assignment/internal/domain/errors"
)

type ItemUsecase interface {
	GetAllItems(ctx context.Context) ([]*entity.Item, error)
	GetItemByID(ctx context.Context, id int64) (*entity.Item, error)
	CreateItem(ctx context.Context, input CreateItemInput) (*entity.Item, error)
	UpdateItem(ctx context.Context, id int64, input UpdateItemInput) (*entity.Item, error) // <-- この行を追加
	DeleteItem(ctx context.Context, id int64) error
	GetCategorySummary(ctx context.Context) (*CategorySummary, error)
}

type CreateItemInput struct {
	Name          string `json:"name"`
	Category      string `json:"category"`
	Brand         string `json:"brand"`
	PurchasePrice int    `json:"purchase_price"`
	PurchaseDate  string `json:"purchase_date"`
}

// UpdateItemInput は PATCH /items/{id} のリクエストボディ
// ポインタ型にして、リクエストに含まれないフィールドがnilになるようにする
type UpdateItemInput struct {
	Name          *string `json:"name,omitempty"`
	Brand         *string `json:"brand,omitempty"`
	PurchasePrice *int    `json:"purchase_price,omitempty"`
}

// --- (既存の CategorySummary, itemUsecase, NewItemUsecase はそのまま) ---

type CategorySummary struct {
	Categories map[string]int `json:"categories"`
	Total      int            `json:"total"`
}

type itemUsecase struct {
	itemRepo ItemRepository
}

func NewItemUsecase(itemRepo ItemRepository) ItemUsecase {
	return &itemUsecase{
		itemRepo: itemRepo,
	}
}

// --- (既存の GetAllItems, GetItemByID, CreateItem はそのまま) ---
func (u *itemUsecase) GetAllItems(ctx context.Context) ([]*entity.Item, error) {
	items, err := u.itemRepo.FindAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve items: %w", err)
	}

	return items, nil
}

func (u *itemUsecase) GetItemByID(ctx context.Context, id int64) (*entity.Item, error) {
	if id <= 0 {
		return nil, domainErrors.ErrInvalidInput
	}

	item, err := u.itemRepo.FindByID(ctx, id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return nil, domainErrors.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to retrieve item: %w", err)
	}

	return item, nil
}

func (u *itemUsecase) CreateItem(ctx context.Context, input CreateItemInput) (*entity.Item, error) {
	// バリデーションして、新しいエンティティを作成
	item, err := entity.NewItem(
		input.Name,
		input.Category,
		input.Brand,
		input.PurchasePrice,
		input.PurchaseDate,
	)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", domainErrors.ErrInvalidInput, err.Error())
	}

	createdItem, err := u.itemRepo.Create(ctx, item)
	if err != nil {
		return nil, fmt.Errorf("failed to create item: %w", err)
	}

	return createdItem, nil
}

// --- UpdateItem メソッドをここに追加 ---
func (u *itemUsecase) UpdateItem(ctx context.Context, id int64, input UpdateItemInput) (*entity.Item, error) {
	if id <= 0 {
		return nil, domainErrors.ErrInvalidInput
	}

	// 1. 既存のアイテムを取得
	existingItem, err := u.itemRepo.FindByID(ctx, id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return nil, domainErrors.ErrItemNotFound
		}
		return nil, fmt.Errorf("failed to retrieve item for update: %w", err)
	}

	// 2. リクエストボディで指定されたフィールドのみバリデーション＆更新
	// (entity.NewItem にあるバリデーションルールを参考に、部分的に適用)
	if input.Name != nil {
		if len(*input.Name) == 0 || len(*input.Name) > 100 {
			return nil, fmt.Errorf("%w: name must be between 1 and 100 characters", domainErrors.ErrInvalidInput)
		}
		existingItem.Name = *input.Name
	}

	if input.Brand != nil {
		if len(*input.Brand) == 0 || len(*input.Brand) > 100 {
			return nil, fmt.Errorf("%w: brand must be between 1 and 100 characters", domainErrors.ErrInvalidInput)
		}
		existingItem.Brand = *input.Brand
	}

	if input.PurchasePrice != nil {
		if *input.PurchasePrice < 0 {
			return nil, fmt.Errorf("%w: purchase_price must be 0 or greater", domainErrors.ErrInvalidInput)
		}
		existingItem.PurchasePrice = *input.PurchasePrice
	}

	// 3. データベースを更新
	// (updated_atはDB側で自動更新される想定)
	updatedItem, err := u.itemRepo.Update(ctx, existingItem)
	if err != nil {
		return nil, fmt.Errorf("failed to update item: %w", err)
	}

	return updatedItem, nil
}

// --- (既存の DeleteItem, GetCategorySummary はそのまま) ---
func (u *itemUsecase) DeleteItem(ctx context.Context, id int64) error {
	if id <= 0 {
		return domainErrors.ErrInvalidInput
	}

	_, err := u.itemRepo.FindByID(ctx, id)
	if err != nil {
		if domainErrors.IsNotFoundError(err) {
			return domainErrors.ErrItemNotFound
		}
		return fmt.Errorf("failed to check item existence: %w", err)
	}

	err = u.itemRepo.Delete(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete item: %w", err)
	}

	return nil
}

func (u *itemUsecase) GetCategorySummary(ctx context.Context) (*CategorySummary, error) {
	categoryCounts, err := u.itemRepo.GetSummaryByCategory(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get category summary: %w", err)
	}

	// 合計計算
	total := 0
	for _, count := range categoryCounts {
		total += count
	}

	summary := make(map[string]int)
	for _, category := range entity.GetValidCategories() {
		if count, exists := categoryCounts[category]; exists {
			summary[category] = count
		} else {
			summary[category] = 0
		}
	}

	return &CategorySummary{
		Categories: summary,
		Total:      total,
	}, nil
}