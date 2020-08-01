package tagrepo

import (
	"github.com/khanhpdt/bookmark-api/internal/app/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Tag struct {
	Name string `json:"name"`
}

type TagList struct {
	List []*Tag `json:"list"`
}

type TagMongoDoc struct {
	Name string `bson:"name"`
}

func SuggestTags() (*TagList, error) {
	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	cursor, err := mongo.TagColl().Find(ctx, bson.M{"bookCount": bson.M{"$gt": 0}})

	if err != nil {
		return nil, err
	}

	decodeCtx, cancelDecodeFunc := mongo.DefaultCtx()
	defer cancelDecodeFunc()

	var tagDocs []TagMongoDoc
	err = cursor.All(decodeCtx, &tagDocs)

	if err != nil {
		return nil, err
	}

	var tags []*Tag
	for _, tagDoc := range tagDocs {
		tag := Tag{Name: tagDoc.Name}
		tags = append(tags, &tag)
	}

	var tagList = TagList{List: []*Tag{}}
	if tags != nil {
		tagList.List = tags
	}
	return &tagList, nil
}

func UpdateTags(currentTags, newTags []string) error {
	toAdd := filter(newTags, func(s string) bool {
		return !include(currentTags, s)
	})

	toRemove := filter(currentTags, func(s string) bool {
		return !include(newTags, s)
	})

	ctx, cancelFunc := mongo.DefaultCtx()
	defer cancelFunc()

	if len(toAdd) > 0 {
		query := bson.M{"name": bson.M{"$in": toAdd}}
		updateObj := bson.M{"$inc": bson.M{"bookCount": 1}}
		upsert := true
		opts := options.UpdateOptions{Upsert: &upsert}

		if _, err := mongo.TagColl().UpdateMany(ctx, query, updateObj, &opts); err != nil {
			return err
		}
	}

	if len(toRemove) > 0 {
		query := bson.M{"name": bson.M{"$in": toRemove}, "bookCount": bson.M{"$gt": 0}}
		updateObj := bson.M{"$inc": bson.M{"bookCount": -1}}

		if _, err := mongo.TagColl().UpdateMany(ctx, query, updateObj); err != nil {
			return err
		}
	}

	return nil
}

func include(arr []string, item string) bool {
	for _, v := range arr {
		if v == item {
			return true
		}
	}
	return false
}

func filter(arr []string, pred func(string) bool) []string {
	res := make([]string, 0)
	for _, v := range arr {
		if pred(v) {
			res = append(res, v)
		}
	}
	return res
}
