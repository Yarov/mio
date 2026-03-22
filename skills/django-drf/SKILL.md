---
name: django-drf
description: >
  Django REST Framework patterns.
  Trigger: When building REST APIs with Django - ViewSets, Serializers, Filters.
metadata:
  author: mio
  version: "1.0"
---

## ViewSet Pattern

```python
from rest_framework import viewsets, status
from rest_framework.response import Response
from rest_framework.decorators import action

class UserViewSet(viewsets.ModelViewSet):
    queryset = User.objects.all()
    serializer_class = UserSerializer
    filterset_class = UserFilter
    permission_classes = [IsAuthenticated]

    def get_serializer_class(self):
        if self.action == "create":
            return UserCreateSerializer
        if self.action in ["update", "partial_update"]:
            return UserUpdateSerializer
        return UserSerializer

    @action(detail=True, methods=["post"])
    def activate(self, request, pk=None):
        user = self.get_object()
        user.is_active = True
        user.save()
        return Response({"status": "activated"})
```

## Serializers

```python
# Read
class UserSerializer(serializers.ModelSerializer):
    full_name = serializers.SerializerMethodField()

    class Meta:
        model = User
        fields = ["id", "email", "full_name", "created_at"]
        read_only_fields = ["id", "created_at"]

    def get_full_name(self, obj):
        return f"{obj.first_name} {obj.last_name}"

# Create (separate serializer)
class UserCreateSerializer(serializers.ModelSerializer):
    password = serializers.CharField(write_only=True)

    class Meta:
        model = User
        fields = ["email", "password", "first_name", "last_name"]

    def create(self, validated_data):
        password = validated_data.pop("password")
        user = User(**validated_data)
        user.set_password(password)
        user.save()
        return user
```

## Filters

```python
from django_filters import rest_framework as filters

class UserFilter(filters.FilterSet):
    email = filters.CharFilter(lookup_expr="icontains")
    is_active = filters.BooleanFilter()
    created_after = filters.DateTimeFilter(field_name="created_at", lookup_expr="gte")

    class Meta:
        model = User
        fields = ["email", "is_active"]
```

## Permissions

```python
class IsOwner(BasePermission):
    def has_object_permission(self, request, view, obj):
        return obj.owner == request.user

class IsAdminOrReadOnly(BasePermission):
    def has_permission(self, request, view):
        if request.method in ["GET", "HEAD", "OPTIONS"]:
            return True
        return request.user.is_staff
```

## Pagination

```python
class StandardPagination(PageNumberPagination):
    page_size = 20
    page_size_query_param = "page_size"
    max_page_size = 100
```

## URL Routing

```python
router = DefaultRouter()
router.register(r"users", UserViewSet, basename="user")
router.register(r"posts", PostViewSet, basename="post")

urlpatterns = [
    path("api/v1/", include(router.urls)),
]
```

## Testing

```python
@pytest.mark.django_db
class TestUserViewSet:
    def test_list_users(self, authenticated_client):
        response = authenticated_client.get("/api/v1/users/")
        assert response.status_code == status.HTTP_200_OK

    def test_create_user(self, authenticated_client):
        data = {"email": "new@test.com", "password": "pass123"}
        response = authenticated_client.post("/api/v1/users/", data)
        assert response.status_code == status.HTTP_201_CREATED
```

## Keywords
django, drf, rest framework, viewset, serializer, api, rest api
